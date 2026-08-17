package core

import (
	"context"
	"encoding/json"
	"testing"
)

// TestExtensionInterfaceSatisfaction verifies that Extension interface has the
// required methods: Name(), Init(ExtensionAPI), Shutdown().
func TestExtensionInterfaceSatisfaction(t *testing.T) {
	// Compile-time check: mockExtension must implement Extension
	var _ Extension = (*mockExtension)(nil)

	ext := &mockExtension{name: "test-ext"}

	// Test Name() returns the extension name
	if got := ext.Name(); got != "test-ext" {
		t.Errorf("Name() = %q, want %q", got, "test-ext")
	}

	// Test Init() receives ExtensionAPI and returns nil
	api := &mockExtensionAPI{}
	if err := ext.Init(api); err != nil {
		t.Errorf("Init() returned error: %v", err)
	}

	// Test Shutdown() returns nil
	if err := ext.Shutdown(); err != nil {
		t.Errorf("Shutdown() returned error: %v", err)
	}
}

// TestExtensionInitError verifies that Init can return an error
func TestExtensionInitError(t *testing.T) {
	ext := &mockExtension{name: "failing-ext", initErr: context.DeadlineExceeded}
	api := &mockExtensionAPI{}

	err := ext.Init(api)
	if err == nil {
		t.Error("Init() should return error, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Init() returned %v, want %v", err, context.DeadlineExceeded)
	}
}

// TestExtensionShutdownError verifies that Shutdown can return an error
func TestExtensionShutdownError(t *testing.T) {
	ext := &mockExtension{name: "shutdown-fail-ext", shutdownErr: context.Canceled}

	err := ext.Shutdown()
	if err == nil {
		t.Error("Shutdown() should return error, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Shutdown() returned %v, want %v", err, context.Canceled)
	}
}

// TestExtensionAPISatisfaction verifies that ExtensionAPI interface has the
// required methods: RegisterTool, RegisterHook, RegisterCommand.
func TestExtensionAPISatisfaction(t *testing.T) {
	// Compile-time check: mockExtensionAPI must implement ExtensionAPI
	var _ ExtensionAPI = (*mockExtensionAPI)(nil)
}

// TestExtensionAPIRegisterTool verifies that RegisterTool accepts a Tool
func TestExtensionAPIRegisterTool(t *testing.T) {
	api := &mockExtensionAPI{}
	tool := &mockTool{name: "test-tool"}

	err := api.RegisterTool(tool)
	if err != nil {
		t.Errorf("RegisterTool() returned error: %v", err)
	}

	if len(api.registeredTools) != 1 {
		t.Errorf("RegisterTool() registered %d tools, want 1", len(api.registeredTools))
	}
	if api.registeredTools[0].Name() != "test-tool" {
		t.Errorf("RegisterTool() registered tool with name %q, want %q", api.registeredTools[0].Name(), "test-tool")
	}
}

// TestExtensionAPIRegisterHook verifies that RegisterHook accepts event and handler
func TestExtensionAPIRegisterHook(t *testing.T) {
	api := &mockExtensionAPI{}
	handler := func(ctx HookContext) error { return nil }

	err := api.RegisterHook("on_turn_start", handler)
	if err != nil {
		t.Errorf("RegisterHook() returned error: %v", err)
	}

	if len(api.registeredHooks) != 1 {
		t.Errorf("RegisterHook() registered %d hooks, want 1", len(api.registeredHooks))
	}
	if _, ok := api.registeredHooks["on_turn_start"]; !ok {
		t.Error("RegisterHook() did not register handler for 'on_turn_start'")
	}
}

// TestExtensionAPIRegisterCommand verifies that RegisterCommand accepts a Command
func TestExtensionAPIRegisterCommand(t *testing.T) {
	api := &mockExtensionAPI{}
	cmd := Command{
		Name:        "test-cmd",
		Description: "A test command",
		Handler: func(ctx context.Context, args string) (string, error) {
			return "result", nil
		},
	}

	err := api.RegisterCommand(cmd)
	if err != nil {
		t.Errorf("RegisterCommand() returned error: %v", err)
	}

	if len(api.registeredCommands) != 1 {
		t.Errorf("RegisterCommand() registered %d commands, want 1", len(api.registeredCommands))
	}
	if api.registeredCommands[0].Name != "test-cmd" {
		t.Errorf("RegisterCommand() registered command with name %q, want %q", api.registeredCommands[0].Name, "test-cmd")
	}
}

// TestCommandHandlerSignature verifies that Command.Handler has the correct signature
func TestCommandHandlerSignature(t *testing.T) {
	called := false
	cmd := Command{
		Name:        "verify-cmd",
		Description: "Verify handler signature",
		Handler: func(ctx context.Context, args string) (string, error) {
			called = true
			return "output", nil
		},
	}

	result, err := cmd.Handler(context.Background(), "--flag")
	if err != nil {
		t.Errorf("Handler() returned error: %v", err)
	}
	if result != "output" {
		t.Errorf("Handler() returned %q, want %q", result, "output")
	}
	if !called {
		t.Error("Handler() was not called")
	}
}

// Mock types for testing

type mockExtension struct {
	name        string
	initErr     error
	shutdownErr error
	api         ExtensionAPI
}

func (m *mockExtension) Name() string { return m.name }

func (m *mockExtension) Init(api ExtensionAPI) error {
	m.api = api
	return m.initErr
}

func (m *mockExtension) Shutdown() error { return m.shutdownErr }

type mockExtensionAPI struct {
	registeredTools    []Tool
	registeredHooks    map[string]HookHandler
	registeredCommands []Command
}

func (m *mockExtensionAPI) RegisterTool(tool Tool) error {
	m.registeredTools = append(m.registeredTools, tool)
	return nil
}

func (m *mockExtensionAPI) RegisterHook(event string, handler HookHandler) error {
	if m.registeredHooks == nil {
		m.registeredHooks = make(map[string]HookHandler)
	}
	m.registeredHooks[event] = handler
	return nil
}

func (m *mockExtensionAPI) RegisterCommand(cmd Command) error {
	m.registeredCommands = append(m.registeredCommands, cmd)
	return nil
}

type mockTool struct {
	name string
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return "mock tool" }
func (m *mockTool) Schema() string      { return "{}" }
func (m *mockTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return "mock result", nil
}
