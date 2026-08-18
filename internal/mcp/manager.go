package mcp

import (
	"context"
	"log"
	"sync"

	"github.com/biggs-100/kui/internal/core"
)

// ClientFactory is a function that creates a Client for a given server configuration.
// In production, this spawns a real subprocess. In tests, it returns mock clients.
type ClientFactory func(ctx context.Context, name string, cfg ServerConfig) (*Client, error)

// defaultClientFactory creates a real client by spawning the subprocess.
func defaultClientFactory(ctx context.Context, name string, cfg ServerConfig) (*Client, error) {
	return NewClient(ctx, cfg.Command, cfg.Cwd, cfg.Environment)
}

// MCPManager manages the lifecycle of all configured MCP servers.
// It provides concurrent connection, tool discovery, and shutdown.
type MCPManager struct {
	config       *Config
	clients      map[string]*Client
	tools        []core.Tool
	clientFactory ClientFactory
	failedCount  int
	mu           sync.Mutex
}

// NewMCPManager creates a new manager with the given configuration.
func NewMCPManager(config *Config) *MCPManager {
	return NewMCPManagerWithFactory(config, defaultClientFactory)
}

// NewMCPManagerWithFactory creates a new manager with a custom client factory.
// This is useful for testing with mock clients.
func NewMCPManagerWithFactory(config *Config, factory ClientFactory) *MCPManager {
	return &MCPManager{
		config:        config,
		clients:       make(map[string]*Client),
		tools:         make([]core.Tool, 0),
		clientFactory: factory,
	}
}

// ConnectAll starts and connects all enabled servers concurrently.
// Server failures are logged and non-fatal per REQ-MCP-15.
func (m *MCPManager) ConnectAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, srv := range m.config.Servers {
		if srv.Disabled {
			log.Printf("mcp manager: skipping disabled server %q", name)
			continue
		}

		wg.Add(1)
		go func(name string, srv ServerConfig) {
			defer wg.Done()

			client, err := m.clientFactory(ctx, name, srv)
			if err != nil {
				log.Printf("mcp manager: failed to start server %q: %v", name, err)
				mu.Lock()
				m.failedCount++
				mu.Unlock()
				return
			}

			if err := client.Initialize(ctx); err != nil {
				log.Printf("mcp manager: failed to initialize server %q: %v", name, err)
				client.Close()
				mu.Lock()
				m.failedCount++
				mu.Unlock()
				return
			}

			// Discover tools
			defs, err := client.ListTools(ctx)
			if err != nil {
				log.Printf("mcp manager: failed to list tools for server %q: %v", name, err)
				client.Close()
				return
			}

			// Create MCPTool instances
			toolList := make([]core.Tool, 0, len(defs))
			for _, def := range defs {
				toolList = append(toolList, NewMCPTool(name, def, client))
			}

			mu.Lock()
			m.clients[name] = client
			m.tools = append(m.tools, toolList...)
			mu.Unlock()

			log.Printf("mcp manager: connected server %q with %d tools", name, len(defs))
		}(name, srv)
	}

	wg.Wait()
	return nil
}

// Status returns the count of connected and failed MCP servers.
func (m *MCPManager) Status() (connected, failed int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.clients), m.failedCount
}

// Shutdown stops all connected servers and cleans up resources.
// It is idempotent per REQ-MCP-13.
func (m *MCPManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, client := range m.clients {
		log.Printf("mcp manager: shutting down server %q", name)
		client.Close()
		delete(m.clients, name)
	}

	m.failedCount = 0
	m.tools = nil
}

// Tools returns all discovered MCP tools as core.Tool implementations.
func (m *MCPManager) Tools() []core.Tool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to avoid races
	result := make([]core.Tool, len(m.tools))
	copy(result, m.tools)
	return result
}
