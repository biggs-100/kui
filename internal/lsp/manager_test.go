package lsp

import (
	"errors"
	"sync"
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// ServerState
// ──────────────────────────────────────────────────────────────────────────────

func TestServerStateString(t *testing.T) {
	tests := []struct {
		state ServerState
		want  string
	}{
		{StateIdle, "idle"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateStopped, "stopped"},
		{StateError, "error"},
		{ServerState(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("ServerState(%d).String() = %q, want %q", int(tt.state), got, tt.want)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// NewLspManager
// ──────────────────────────────────────────────────────────────────────────────

func TestNewLspManager(t *testing.T) {
	m := NewLspManager()
	if m == nil {
		t.Fatal("NewLspManager() returned nil")
	}
	if m.Servers() != 0 {
		t.Errorf("Servers() = %d, want 0", m.Servers())
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// IsRunning
// ──────────────────────────────────────────────────────────────────────────────

func TestIsRunningNotStarted(t *testing.T) {
	m := NewLspManager()
	if m.IsRunning("file:///tmp/project") {
		t.Error("IsRunning should be false before any server starts")
	}
}

func TestIsRunningAfterManualStart(t *testing.T) {
	m := NewLspManager()
	root := "file:///tmp/project"

	// Start with a mock-friendly approach: use StartServer with a fake path
	// We expect this to fail (no real server), but we can test the state machine
	err := m.StartServer(root, "nonexistent-server-binary", nil)
	// This should fail since the binary doesn't exist
	if err == nil {
		// If it somehow succeeded, clean up
		m.StopServer(root)
		t.Skip("server started unexpectedly")
	}

	// After failed start, state should be error or stopped
	if m.IsRunning(root) {
		t.Error("IsRunning should be false after failed start")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// StopServer
// ──────────────────────────────────────────────────────────────────────────────

func TestStopServerNotRunning(t *testing.T) {
	m := NewLspManager()
	err := m.StopServer("file:///tmp/project")
	// Should not panic or error — stopping a non-existent server is a no-op
	if err != nil {
		t.Errorf("StopServer() on non-existent server should not error, got: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// StopAll
// ──────────────────────────────────────────────────────────────────────────────

func TestStopAllEmpty(t *testing.T) {
	m := NewLspManager()
	err := m.StopAll()
	if err != nil {
		t.Errorf("StopAll() on empty manager should not error, got: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// GetServer with mock client
// ──────────────────────────────────────────────────────────────────────────────

func TestGetServerReturnsSameClient(t *testing.T) {
	m := NewLspManager()
	root := "file:///tmp/project"

	// Inject a mock client directly
	_, mockClient, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer mockClient.Stop()

	m.mu.Lock()
	m.servers[root] = &serverEntry{
		client: mockClient,
		state:  StateRunning,
	}
	m.mu.Unlock()

	client1, err := m.GetServer(root)
	if err != nil {
		t.Fatalf("GetServer() error: %v", err)
	}
	client2, err := m.GetServer(root)
	if err != nil {
		t.Fatalf("GetServer() error: %v", err)
	}
	if client1 != client2 {
		t.Error("GetServer() should return the same client for the same root")
	}
}

func TestGetServerDifferentRoots(t *testing.T) {
	m := NewLspManager()

	// Inject two mock clients for different roots
	_, client1, _ := NewMockLspServer()
	defer client1.Stop()
	_, client2, _ := NewMockLspServer()
	defer client2.Stop()

	m.mu.Lock()
	m.servers["file:///project1"] = &serverEntry{client: client1, state: StateRunning}
	m.servers["file:///project2"] = &serverEntry{client: client2, state: StateRunning}
	m.mu.Unlock()

	got1, err := m.GetServer("file:///project1")
	if err != nil {
		t.Fatalf("GetServer() error: %v", err)
	}
	got2, err := m.GetServer("file:///project2")
	if err != nil {
		t.Fatalf("GetServer() error: %v", err)
	}
	if got1 == got2 {
		t.Error("different roots should return different clients")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Servers count
// ──────────────────────────────────────────────────────────────────────────────

func TestServersCount(t *testing.T) {
	m := NewLspManager()

	_, c1, _ := NewMockLspServer()
	defer c1.Stop()
	_, c2, _ := NewMockLspServer()
	defer c2.Stop()

	m.mu.Lock()
	m.servers["file:///a"] = &serverEntry{client: c1, state: StateRunning}
	m.servers["file:///b"] = &serverEntry{client: c2, state: StateRunning}
	m.mu.Unlock()

	if got := m.Servers(); got != 2 {
		t.Errorf("Servers() = %d, want 2", got)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// State
// ──────────────────────────────────────────────────────────────────────────────

func TestStateForUnknownRoot(t *testing.T) {
	m := NewLspManager()
	state := m.State("file:///unknown")
	if state != StateIdle {
		t.Errorf("State() for unknown root = %v, want StateIdle", state)
	}
}

func TestStateForKnownRoot(t *testing.T) {
	m := NewLspManager()

	_, c, _ := NewMockLspServer()
	defer c.Stop()

	m.mu.Lock()
	m.servers["file:///project"] = &serverEntry{client: c, state: StateRunning}
	m.mu.Unlock()

	state := m.State("file:///project")
	if state != StateRunning {
		t.Errorf("State() = %v, want StateRunning", state)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Auto-restart (crashed server → next GetServer triggers restart)
// ──────────────────────────────────────────────────────────────────────────────

func TestAutoRestartAfterCrash(t *testing.T) {
	m := NewLspManager()
	root := "file:///tmp/project"

	// Inject a server entry in error state
	_, c, _ := NewMockLspServer()
	defer c.Stop()

	m.mu.Lock()
	m.servers[root] = &serverEntry{client: c, state: StateError}
	m.mu.Unlock()

	// GetServer on an error-state entry should attempt restart
	// (which will fail because no real binary, but the attempt proves the path works)
	_, err := m.GetServer(root)
	if err == nil {
		t.Error("GetServer on error-state server should return an error when no binary configured")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Lazy startup via ensureRunning
// ──────────────────────────────────────────────────────────────────────────────

func TestGetServerNoDefaultReturnsNotReady(t *testing.T) {
	m := NewLspManager()
	// No default server configured — should return ServerNotReadyError.
	_, err := m.GetServer("file:///unknown")
	if err == nil {
		t.Fatal("GetServer on unknown root without default should error")
	}
	var nre *ServerNotReadyError
	if !errors.As(err, &nre) {
		t.Errorf("expected ServerNotReadyError, got %T: %v", err, err)
	}
}

func TestGetServerWithDefaultTriggersLazyStartup(t *testing.T) {
	m := NewLspManager()
	// Configure a default server path that doesn't exist — startup will fail,
	// but the attempt proves ensureRunning was triggered.
	m.SetDefaultServer("nonexistent-server-binary", nil)

	_, err := m.GetServer("file:///new-project")
	if err == nil {
		t.Fatal("GetServer with nonexistent binary should error")
	}
	// After failed startup, an entry should exist in error state.
	if m.State("file:///new-project") != StateError {
		t.Errorf("State after failed lazy startup = %v, want StateError", m.State("file:///new-project"))
	}
}

func TestGetServerExistingRunningReturnsClient(t *testing.T) {
	m := NewLspManager()
	root := "file:///project"

	_, c, _ := NewMockLspServer()
	defer c.Stop()

	m.mu.Lock()
	m.servers[root] = &serverEntry{client: c, state: StateRunning}
	m.mu.Unlock()

	client, err := m.GetServer(root)
	if err != nil {
		t.Fatalf("GetServer on running entry error: %v", err)
	}
	if client != c {
		t.Error("GetServer should return the existing running client")
	}
}

func TestSetDefaultServer(t *testing.T) {
	m := NewLspManager()
	m.SetDefaultServer("/usr/bin/gopls", []string{"-remote=auto"})

	m.mu.Lock()
	path := m.defaultServerPath
	args := m.defaultArgs
	m.mu.Unlock()

	if path != "/usr/bin/gopls" {
		t.Errorf("defaultServerPath = %q, want %q", path, "/usr/bin/gopls")
	}
	if len(args) != 1 || args[0] != "-remote=auto" {
		t.Errorf("defaultArgs = %v, want [-remote=auto]", args)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Concurrent access
// ──────────────────────────────────────────────────────────────────────────────

func TestManagerConcurrentAccess(t *testing.T) {
	m := NewLspManager()
	var wg sync.WaitGroup

	// Concurrently insert and query servers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			root := "file:///project" + string(rune('A'+idx))
			_, c, _ := NewMockLspServer()
			defer c.Stop()

			m.mu.Lock()
			m.servers[root] = &serverEntry{client: c, state: StateRunning}
			m.mu.Unlock()

			_ = m.IsRunning(root)
			_ = m.State(root)
			_ = m.Servers()
		}(i)
	}
	wg.Wait()

	if m.Servers() != 10 {
		t.Errorf("Servers() = %d, want 10", m.Servers())
	}
}
