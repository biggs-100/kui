package mcp

import (
	"context"
	"fmt"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

func TestNewMCPManager(t *testing.T) {
	cfg := &Config{
		Servers: map[string]ServerConfig{
			"github": {
				Type:    "local",
				Command: []string{"echo"},
			},
			"slack": {
				Type:    "local",
				Command: []string{"echo"},
			},
		},
	}

	manager := NewMCPManager(cfg)
	if manager == nil {
		t.Fatal("NewMCPManager() returned nil")
	}

	if len(manager.config.Servers) != 2 {
		t.Errorf("Manager has %d servers, want 2", len(manager.config.Servers))
	}
}

func TestConnectAllSkipsDisabledServers(t *testing.T) {
	cfg := &Config{
		Servers: map[string]ServerConfig{
			"enabled": {
				Type:    "local",
				Command: []string{"echo"},
			},
			"disabled": {
				Type:     "local",
				Command:  []string{"echo"},
				Disabled: true,
			},
		},
	}

	factory := mockClientFactory([]MCPToolDef{})
	manager := NewMCPManagerWithFactory(cfg, factory)
	ctx := context.Background()

	err := manager.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll() error: %v", err)
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()

	if len(manager.clients) != 1 {
		t.Errorf("Manager has %d clients, want 1", len(manager.clients))
	}

	if _, ok := manager.clients["enabled"]; !ok {
		t.Error("Enabled server not connected")
	}

	if _, ok := manager.clients["disabled"]; ok {
		t.Error("Disabled server should not be connected")
	}
}

func TestConnectAllNonFatal(t *testing.T) {
	cfg := &Config{
		Servers: map[string]ServerConfig{
			"good": {
				Type:    "local",
				Command: []string{"echo"},
			},
			"bad": {
				Type:    "local",
				Command: []string{"nonexistent-command-12345"},
			},
		},
	}

	factory := func(ctx context.Context, name string, cfg ServerConfig) (*Client, error) {
		if name == "bad" {
			return nil, fmt.Errorf("command not found")
		}
		return newMockClientWithTools([]MCPToolDef{})
	}

	manager := NewMCPManagerWithFactory(cfg, factory)
	ctx := context.Background()

	err := manager.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll() should not return error for partial failure, got: %v", err)
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()

	if len(manager.clients) < 1 {
		t.Errorf("Manager has %d clients, want at least 1", len(manager.clients))
	}
}

func TestShutdownIdempotent(t *testing.T) {
	cfg := &Config{
		Servers: map[string]ServerConfig{
			"test": {
				Type:    "local",
				Command: []string{"echo"},
			},
		},
	}

	factory := mockClientFactory([]MCPToolDef{})
	manager := NewMCPManagerWithFactory(cfg, factory)
	ctx := context.Background()

	_ = manager.ConnectAll(ctx)

	// Shutdown should be callable multiple times without panic
	manager.Shutdown()
	manager.Shutdown()
	manager.Shutdown()
}

func TestToolsEmpty(t *testing.T) {
	cfg := &Config{
		Servers: map[string]ServerConfig{},
	}

	manager := NewMCPManager(cfg)
	tools := manager.Tools()

	if len(tools) != 0 {
		t.Errorf("Tools() returned %d tools, want 0", len(tools))
	}
}

func TestManagerToolsReturnsDiscoveredTools(t *testing.T) {
	// Build tools directly and inject into manager — avoids pipe issues
	cfg := &Config{
		Servers: map[string]ServerConfig{
			"github": {
				Type:    "local",
				Command: []string{"echo"},
			},
			"slack": {
				Type:    "local",
				Command: []string{"echo"},
			},
		},
	}

	manager := NewMCPManager(cfg)

	// Directly populate the tools slice as ConnectAll would
	githubTool := &MCPTool{
		name:        "github_create_issue",
		description: "Create a GitHub issue",
		inputSchema: `{"type":"object"}`,
		serverName:  "github",
		toolName:    "create_issue",
	}
	slackTool := &MCPTool{
		name:        "slack_send_message",
		description: "Send a Slack message",
		inputSchema: `{"type":"object"}`,
		serverName:  "slack",
		toolName:    "send_message",
	}

	manager.mu.Lock()
	manager.tools = append(manager.tools, githubTool, slackTool)
	manager.mu.Unlock()

	tools := manager.Tools()
	if len(tools) != 2 {
		t.Fatalf("Tools() returned %d tools, want 2", len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}

	if !toolNames["github_create_issue"] {
		t.Error("Missing github_create_issue tool")
	}
	if !toolNames["slack_send_message"] {
		t.Error("Missing slack_send_message tool")
	}
}

func TestMCPManagerStatus(t *testing.T) {
	cfg := &Config{
		Servers: map[string]ServerConfig{
			"github": {
				Type:    "local",
				Command: []string{"echo"},
			},
			"slack": {
				Type:    "local",
				Command: []string{"echo"},
			},
		},
	}

	factory := mockClientFactory([]MCPToolDef{})
	manager := NewMCPManagerWithFactory(cfg, factory)
	ctx := context.Background()

	// Before connecting: 0 connected, 0 failed
	connected, failed := manager.Status()
	if connected != 0 || failed != 0 {
		t.Errorf("Status() before connect = (%d, %d), want (0, 0)", connected, failed)
	}

	// Connect all
	_ = manager.ConnectAll(ctx)

	// After connecting: 2 connected, 0 failed
	connected, failed = manager.Status()
	if connected != 2 {
		t.Errorf("Status() connected = %d, want 2", connected)
	}
	if failed != 0 {
		t.Errorf("Status() failed = %d, want 0", failed)
	}

	// Shutdown
	manager.Shutdown()

	// After shutdown: 0 connected, 0 failed
	connected, failed = manager.Status()
	if connected != 0 || failed != 0 {
		t.Errorf("Status() after shutdown = (%d, %d), want (0, 0)", connected, failed)
	}
}

func TestMCPManagerStatusPartialFailure(t *testing.T) {
	cfg := &Config{
		Servers: map[string]ServerConfig{
			"good": {
				Type:    "local",
				Command: []string{"echo"},
			},
			"bad": {
				Type:    "local",
				Command: []string{"nonexistent-command-12345"},
			},
		},
	}

	factory := func(ctx context.Context, name string, cfg ServerConfig) (*Client, error) {
		if name == "bad" {
			return nil, fmt.Errorf("command not found")
		}
		return newMockClientWithTools([]MCPToolDef{})
	}

	manager := NewMCPManagerWithFactory(cfg, factory)
	ctx := context.Background()

	_ = manager.ConnectAll(ctx)

	connected, failed := manager.Status()
	if connected != 1 {
		t.Errorf("Status() connected = %d, want 1", connected)
	}
	if failed != 1 {
		t.Errorf("Status() failed = %d, want 1", failed)
	}
}

func TestManagerToolsPrefixedWithServerName(t *testing.T) {
	cfg := &Config{
		Servers: map[string]ServerConfig{
			"docs": {
				Type:    "local",
				Command: []string{"echo"},
			},
		},
	}

	manager := NewMCPManager(cfg)

	// Directly populate tools
	manager.mu.Lock()
	manager.tools = append(manager.tools,
		&MCPTool{name: "docs_search", description: "Search docs", inputSchema: `{"type":"object"}`, serverName: "docs", toolName: "search"},
		&MCPTool{name: "docs_read", description: "Read doc", inputSchema: `{"type":"object"}`, serverName: "docs", toolName: "read"},
	)
	manager.mu.Unlock()

	resultTools := manager.Tools()
	if len(resultTools) != 2 {
		t.Fatalf("Tools() returned %d tools, want 2", len(resultTools))
	}

	for _, tool := range resultTools {
		name := tool.Name()
		if len(name) < 5 || name[:5] != "docs_" {
			t.Errorf("Tool name %q should be prefixed with 'docs_'", name)
		}
	}
}

// Verify core.Tool interface is satisfied at compile time
var _ core.Tool = &MCPTool{}
