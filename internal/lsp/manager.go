package lsp

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
)

// ServerState represents the state of an LSP server.
type ServerState int

const (
	StateIdle     ServerState = iota // Not started
	StateStarting                    // Handshake in progress
	StateRunning                     // Ready to serve requests
	StateStopped                     // Gracefully shut down
	StateError                       // Crashed or failed to start
)

// String returns the human-readable name for a ServerState.
func (s ServerState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopped:
		return "stopped"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// serverEntry tracks a single LSP server instance.
type serverEntry struct {
	client     *LspClient
	state      ServerState
	serverPath string
	args       []string
}

// LspManager manages multiple LSP server instances keyed by workspace root URI.
// One gopls instance per workspace. Supports lazy startup and auto-restart.
type LspManager struct {
	mu      sync.Mutex
	servers map[string]*serverEntry
	cache   *DiagnosticCache

	// defaultServerPath is the LSP server binary path used for lazy startup.
	// When set, GetServer() can auto-start servers for unknown root URIs.
	defaultServerPath string
	defaultArgs       []string
}

// NewLspManager creates a new LspManager.
func NewLspManager() *LspManager {
	return &LspManager{
		servers: make(map[string]*serverEntry),
		cache:   NewDiagnosticCache(),
	}
}

// SetDefaultServer configures the LSP server binary and args for lazy startup.
// When set, GetServer() will auto-start a server for unknown root URIs using
// these defaults.
func (m *LspManager) SetDefaultServer(serverPath string, args []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultServerPath = serverPath
	m.defaultArgs = args
}

// ensureRunning starts a server entry if the state is Idle, Error, or if no
// entry exists and a default server path is configured. Returns the client
// or an error if startup fails.
func (m *LspManager) ensureRunning(rootUri string) (*LspClient, error) {
	m.mu.Lock()
	entry, exists := m.servers[rootUri]
	defaultPath := m.defaultServerPath
	defaultArgs := m.defaultArgs
	m.mu.Unlock()

	if exists {
		switch entry.state {
		case StateRunning:
			return entry.client, nil
		case StateStarting:
			return nil, fmt.Errorf("lsp server for %s is still starting", rootUri)
		case StateStopped, StateIdle, StateError:
			return m.startServerEntry(rootUri, entry)
		}
	}

	// No entry — try lazy startup if we have a default server path.
	if defaultPath != "" {
		entry = &serverEntry{
			serverPath: defaultPath,
			args:       defaultArgs,
			state:      StateIdle,
		}
		m.mu.Lock()
		m.servers[rootUri] = entry
		m.mu.Unlock()
		return m.startServerEntry(rootUri, entry)
	}

	return nil, &ServerNotReadyError{Tool: "lsp"}
}

// GetServer returns the LSP client for the given workspace root URI.
// If no server exists, it attempts lazy startup using the stored server path.
// If the server crashed, it triggers auto-restart.
func (m *LspManager) GetServer(rootUri string) (*LspClient, error) {
	return m.ensureRunning(rootUri)
}

// StartServer starts an LSP server for the given workspace root.
func (m *LspManager) StartServer(rootUri string, serverPath string, args []string) error {
	m.mu.Lock()
	if entry, exists := m.servers[rootUri]; exists {
		if entry.state == StateRunning {
			m.mu.Unlock()
			return nil // already running
		}
	}
	m.mu.Unlock()

	entry := &serverEntry{
		serverPath: serverPath,
		args:       args,
		state:      StateStarting,
	}

	m.mu.Lock()
	m.servers[rootUri] = entry
	m.mu.Unlock()

	_, err := m.startServerEntry(rootUri, entry)
	return err
}

// StopServer stops the LSP server for the given workspace root.
func (m *LspManager) StopServer(rootUri string) error {
	m.mu.Lock()
	entry, exists := m.servers[rootUri]
	if !exists {
		m.mu.Unlock()
		return nil
	}
	entry.state = StateStopped
	m.mu.Unlock()

	if entry.client != nil {
		return entry.client.Stop()
	}
	return nil
}

// StopAll stops all running LSP servers.
func (m *LspManager) StopAll() error {
	m.mu.Lock()
	entries := make([]*serverEntry, 0, len(m.servers))
	for _, entry := range m.servers {
		entries = append(entries, entry)
	}
	m.mu.Unlock()

	var lastErr error
	for _, entry := range entries {
		if entry.state == StateRunning && entry.client != nil {
			if err := entry.client.Stop(); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

// IsRunning returns true if a server for the given root is in the Running state.
func (m *LspManager) IsRunning(rootUri string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, exists := m.servers[rootUri]
	return exists && entry.state == StateRunning
}

// State returns the current state of the server for the given root.
// Returns StateIdle if no server entry exists.
func (m *LspManager) State(rootUri string) ServerState {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, exists := m.servers[rootUri]
	if !exists {
		return StateIdle
	}
	return entry.state
}

// Servers returns the number of tracked server entries.
func (m *LspManager) Servers() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.servers)
}

// Cache returns the shared diagnostic cache.
func (m *LspManager) Cache() *DiagnosticCache {
	return m.cache
}

// startServerEntry starts the server described by the entry and updates state.
func (m *LspManager) startServerEntry(rootUri string, entry *serverEntry) (*LspClient, error) {
	m.mu.Lock()
	entry.state = StateStarting
	m.mu.Unlock()

	// Validate server binary exists
	if _, err := exec.LookPath(entry.serverPath); err != nil {
		m.mu.Lock()
		entry.state = StateError
		m.mu.Unlock()
		return nil, fmt.Errorf("lsp server %q not found in PATH: %w", entry.serverPath, err)
	}

	client, err := NewLspClient(entry.serverPath, entry.args, rootUri)
	if err != nil {
		m.mu.Lock()
		entry.state = StateError
		m.mu.Unlock()
		return nil, fmt.Errorf("lsp client create: %w", err)
	}

	if err := client.Start(); err != nil {
		m.mu.Lock()
		entry.state = StateError
		m.mu.Unlock()
		return nil, fmt.Errorf("lsp client start: %w", err)
	}

	if err := client.Initialize(rootUri); err != nil {
		m.mu.Lock()
		entry.state = StateError
		m.mu.Unlock()
		return nil, fmt.Errorf("lsp initialize: %w", err)
	}

	// Wire up push-based diagnostics to the cache
	client.SetNotificationHandler(func(method string, params json.RawMessage) {
		if method == "textDocument/publishDiagnostics" {
			var notif struct {
				URI         string       `json:"uri"`
				Diagnostics []Diagnostic `json:"diagnostics"`
			}
			if err := json.Unmarshal(params, &notif); err == nil {
				m.cache.Set(notif.URI, notif.Diagnostics)
			}
		}
	})

	m.mu.Lock()
	entry.client = client
	entry.state = StateRunning
	m.mu.Unlock()

	return client, nil
}

// restartServer stops the old client and starts a new one.
func (m *LspManager) restartServer(rootUri string, entry *serverEntry) (*LspClient, error) {
	// Stop old client if still around
	if entry.client != nil {
		_ = entry.client.Stop()
	}
	m.cache.ClearAll()
	return m.startServerEntry(rootUri, entry)
}
