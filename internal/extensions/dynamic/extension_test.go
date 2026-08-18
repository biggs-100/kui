package dynamic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

// --- Mock ClientInterface ---

// mockClient implements ClientInterface for unit tests without subprocesses.
type mockClient struct {
	tools       []ToolDef
	callResult  string
	callErr     error
	initErr     error
	listErr     error
	closed      bool
}

func (m *mockClient) Initialize(_ context.Context) error {
	return m.initErr
}

func (m *mockClient) ListTools(_ context.Context) ([]ToolDef, error) {
	return m.tools, m.listErr
}

func (m *mockClient) CallTool(_ context.Context, name string, _ json.RawMessage) (string, error) {
	if m.callErr != nil {
		return "", m.callErr
	}
	return m.callResult, nil
}

func (m *mockClient) Shutdown(_ context.Context) error {
	m.closed = true
	return nil
}

// --- mockExtensionAPI implements core.ExtensionAPI ---

type mockExtensionAPI struct {
	tools []core.Tool
}

func (m *mockExtensionAPI) RegisterTool(tool core.Tool) error {
	m.tools = append(m.tools, tool)
	return nil
}

func (m *mockExtensionAPI) RegisterHook(_ string, _ core.HookHandler) error {
	return nil
}

func (m *mockExtensionAPI) RegisterCommand(_ core.Command) error {
	return nil
}

// --- Tests ---

func TestDynamicToolName(t *testing.T) {
	mock := &mockClient{}
	def := ToolDef{Name: "read_file", Description: "Read a file", InputSchema: json.RawMessage(`{"type":"object"}`)}
	tool := NewDynamicTool("myext", def, mock)

	if tool.Name() != "myext_read_file" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "myext_read_file")
	}
	if tool.Description() != "Read a file" {
		t.Errorf("Description() = %q, want %q", tool.Description(), "Read a file")
	}
	if tool.Schema() != `{"type":"object"}` {
		t.Errorf("Schema() = %q, want %q", tool.Schema(), `{"type":"object"}`)
	}
}

func TestDynamicToolExecute(t *testing.T) {
	mock := &mockClient{callResult: "hello world"}
	def := ToolDef{Name: "greet", Description: "Say hello", InputSchema: json.RawMessage(`{}`)}
	tool := NewDynamicTool("myext", def, mock)

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"name":"kui"}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "hello world" {
		t.Errorf("Execute() = %q, want %q", result, "hello world")
	}
}

func TestDynamicExtensionInitRegistersTools(t *testing.T) {
	mock := &mockClient{
		tools: []ToolDef{
			{Name: "foo", Description: "foo desc", InputSchema: json.RawMessage(`{"type":"object","properties":{}}`)},
			{Name: "bar", Description: "bar desc", InputSchema: json.RawMessage(`{"type":"object","properties":{}}`)},
		},
	}
	api := &mockExtensionAPI{}
	factory := func(_ context.Context, _ string) (ClientInterface, error) {
		return mock, nil
	}

	ext := NewDynamicExtension(&Manifest{Name: "myext", EntryPoint: "/usr/bin/ext"}, factory)
	if err := ext.Init(api); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if len(api.tools) != 2 {
		t.Fatalf("got %d registered tools, want 2", len(api.tools))
	}
	if api.tools[0].Name() != "myext_foo" {
		t.Errorf("tool[0].Name() = %q, want %q", api.tools[0].Name(), "myext_foo")
	}
	if api.tools[1].Name() != "myext_bar" {
		t.Errorf("tool[1].Name() = %q, want %q", api.tools[1].Name(), "myext_bar")
	}
}

func TestDynamicExtensionInitSpawnError(t *testing.T) {
	factory := func(_ context.Context, _ string) (ClientInterface, error) {
		return nil, errors.New("executable not found")
	}
	api := &mockExtensionAPI{}

	ext := NewDynamicExtension(&Manifest{Name: "badext", EntryPoint: "/no/such/binary"}, factory)
	err := ext.Init(api)
	if err == nil {
		t.Fatal("expected error for spawn failure, got nil")
	}

	var spawnErr *SpawnError
	if !errors.As(err, &spawnErr) {
		t.Fatalf("expected SpawnError, got: %v", err)
	}
	if spawnErr.Extension != "badext" {
		t.Errorf("SpawnError.Extension = %q, want %q", spawnErr.Extension, "badext")
	}
	// No tools should be registered.
	if len(api.tools) != 0 {
		t.Errorf("got %d registered tools, want 0", len(api.tools))
	}
}

func TestDynamicExtensionInitHandshakeError(t *testing.T) {
	mock := &mockClient{initErr: errors.New("handshake timeout")}
	factory := func(_ context.Context, _ string) (ClientInterface, error) {
		return mock, nil
	}
	api := &mockExtensionAPI{}

	ext := NewDynamicExtension(&Manifest{Name: "timeout", EntryPoint: "/usr/bin/ext"}, factory)
	err := ext.Init(api)
	if err == nil {
		t.Fatal("expected error for handshake failure, got nil")
	}

	var protoErr *ProtocolError
	if !errors.As(err, &protoErr) {
		t.Fatalf("expected ProtocolError, got: %v", err)
	}
	if protoErr.Method != "initialize" {
		t.Errorf("ProtocolError.Method = %q, want %q", protoErr.Method, "initialize")
	}
}

func TestDynamicExtensionInitListToolsError(t *testing.T) {
	mock := &mockClient{listErr: errors.New("list failed")}
	factory := func(_ context.Context, _ string) (ClientInterface, error) {
		return mock, nil
	}
	api := &mockExtensionAPI{}

	ext := NewDynamicExtension(&Manifest{Name: "listfail", EntryPoint: "/usr/bin/ext"}, factory)
	err := ext.Init(api)
	if err == nil {
		t.Fatal("expected error for list tools failure, got nil")
	}
	if !strings.Contains(err.Error(), "list tools") {
		t.Errorf("error = %q, want it to contain 'list tools'", err.Error())
	}
}

func TestDynamicExtensionInitRegisterToolError(t *testing.T) {
	mock := &mockClient{
		tools: []ToolDef{
			{Name: "dup", Description: "dup tool", InputSchema: json.RawMessage(`{}`)},
		},
	}
	// API that rejects duplicates.
	api := &failingExtensionAPI{err: &core.DuplicateToolError{Name: "dup"}}
	factory := func(_ context.Context, _ string) (ClientInterface, error) {
		return mock, nil
	}

	ext := NewDynamicExtension(&Manifest{Name: "dupext", EntryPoint: "/usr/bin/ext"}, factory)
	err := ext.Init(api)
	if err == nil {
		t.Fatal("expected error for register tool failure, got nil")
	}
	if !strings.Contains(err.Error(), "register tool") {
		t.Errorf("error = %q, want it to contain 'register tool'", err.Error())
	}
}

func TestDynamicExtensionShutdownNilClient(t *testing.T) {
	ext := NewDynamicExtension(&Manifest{Name: "neverinit", EntryPoint: "/usr/bin/ext"}, nil)
	// Shutdown before Init should not panic.
	if err := ext.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestDynamicExtensionCrashDoesNotRegisterTools(t *testing.T) {
	factory := func(_ context.Context, _ string) (ClientInterface, error) {
		return nil, errors.New("process crashed")
	}
	api := &mockExtensionAPI{}

	ext := NewDynamicExtension(&Manifest{Name: "crash", EntryPoint: "/usr/bin/crash"}, factory)
	err := ext.Init(api)
	if err == nil {
		t.Fatal("expected error for crash, got nil")
	}
	// Manager must not register any tools from a crashed extension.
	if len(api.tools) != 0 {
		t.Errorf("got %d registered tools, want 0", len(api.tools))
	}
}

// --- Helpers ---

// failingExtensionAPI rejects RegisterTool with a preset error.
type failingExtensionAPI struct {
	err error
}

func (m *failingExtensionAPI) RegisterTool(_ core.Tool) error { return m.err }
func (m *failingExtensionAPI) RegisterHook(_ string, _ core.HookHandler) error {
	return nil
}
func (m *failingExtensionAPI) RegisterCommand(_ core.Command) error { return nil }
