package lsp

import (
	"context"
	"encoding/json"
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// Mock ToolManager for tool tests
// ──────────────────────────────────────────────────────────────────────────────

// toolManager is the interface tools depend on for testing.
type toolManager interface {
	GetServer(rootUri string) (*LspClient, error)
	IsRunning(rootUri string) bool
	Cache() *DiagnosticCache
}

// mockToolManager is a test double that returns injected clients and cache.
type mockToolManager struct {
	clients map[string]*LspClient
	cache   *DiagnosticCache
}

func newMockToolManager() *mockToolManager {
	return &mockToolManager{
		clients: make(map[string]*LspClient),
		cache:   NewDiagnosticCache(),
	}
}

func (m *mockToolManager) GetServer(rootUri string) (*LspClient, error) {
	c, ok := m.clients[rootUri]
	if !ok {
		return nil, &ServerNotReadyError{Tool: "test"}
	}
	return c, nil
}

func (m *mockToolManager) IsRunning(rootUri string) bool {
	_, ok := m.clients[rootUri]
	return ok
}

func (m *mockToolManager) Cache() *DiagnosticCache {
	return m.cache
}

// ──────────────────────────────────────────────────────────────────────────────
// lsp_diagnostics tool
// ──────────────────────────────────────────────────────────────────────────────

func TestLspDiagnosticsToolName(t *testing.T) {
	mgr := newMockToolManager()
	tool := NewLspDiagnosticsTool(mgr)
	if tool.Name() != "lsp_diagnostics" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "lsp_diagnostics")
	}
}

func TestLspDiagnosticsToolReturnsCacheData(t *testing.T) {
	mgr := newMockToolManager()
	uri := "file:///tmp/test.go"

	// Seed cache with diagnostics
	mgr.cache.Set(uri, []Diagnostic{
		{
			Range:    Range{Start: Position{Line: 1, Character: 0}, End: Position{Line: 1, Character: 5}},
			Severity: DiagnosticSeverityError,
			Message:  "undefined: foo",
		},
	})

	tool := NewLspDiagnosticsTool(mgr)
	params, _ := json.Marshal(map[string]interface{}{"uri": uri})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var out struct {
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(out.Diagnostics) != 1 {
		t.Fatalf("diagnostics len = %d, want 1", len(out.Diagnostics))
	}
	if out.Diagnostics[0].Message != "undefined: foo" {
		t.Errorf("message = %q, want %q", out.Diagnostics[0].Message, "undefined: foo")
	}
}

func TestLspDiagnosticsToolEmpty(t *testing.T) {
	mgr := newMockToolManager()
	tool := NewLspDiagnosticsTool(mgr)
	params, _ := json.Marshal(map[string]interface{}{"uri": "file:///tmp/unknown.go"})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var out struct {
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(out.Diagnostics) != 0 {
		t.Errorf("expected empty diagnostics, got %d", len(out.Diagnostics))
	}
}

func TestLspDiagnosticsToolMissingURI(t *testing.T) {
	mgr := newMockToolManager()
	tool := NewLspDiagnosticsTool(mgr)
	params, _ := json.Marshal(map[string]interface{}{})

	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Error("expected error for missing uri, got nil")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// lsp_hover tool
// ──────────────────────────────────────────────────────────────────────────────

func TestLspHoverToolName(t *testing.T) {
	mgr := newMockToolManager()
	tool := NewLspHoverTool(mgr)
	if tool.Name() != "lsp_hover" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "lsp_hover")
	}
}

func TestLspHoverToolDelegatesToClient(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	server.Handle("textDocument/hover", func(params json.RawMessage) interface{} {
		return Hover{Contents: "func main()"}
	})

	mgr := newMockToolManager()
	mgr.clients["file:///tmp/test.go"] = client

	tool := NewLspHoverTool(mgr)
	params, _ := json.Marshal(map[string]interface{}{
		"uri":       "file:///tmp/test.go",
		"line":      5,
		"character": 5,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var out struct {
		Contents string `json:"contents"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if out.Contents != "func main()" {
		t.Errorf("contents = %q, want %q", out.Contents, "func main()")
	}
}

func TestLspHoverToolServerNotRunning(t *testing.T) {
	mgr := newMockToolManager()
	tool := NewLspHoverTool(mgr)
	params, _ := json.Marshal(map[string]interface{}{
		"uri":       "file:///tmp/test.go",
		"line":      0,
		"character": 0,
	})

	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Error("expected error when server not running, got nil")
	}
}

func TestLspHoverToolMissingParams(t *testing.T) {
	mgr := newMockToolManager()
	tool := NewLspHoverTool(mgr)
	params, _ := json.Marshal(map[string]interface{}{})

	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Error("expected error for missing params, got nil")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// lsp_definition tool
// ──────────────────────────────────────────────────────────────────────────────

func TestLspDefinitionToolName(t *testing.T) {
	mgr := newMockToolManager()
	tool := NewLspDefinitionTool(mgr)
	if tool.Name() != "lsp_definition" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "lsp_definition")
	}
}

func TestLspDefinitionToolDelegatesToClient(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	server.Handle("textDocument/definition", func(params json.RawMessage) interface{} {
		return []Location{
			{
				URI:   "file:///tmp/other.go",
				Range: Range{Start: Position{Line: 10, Character: 0}, End: Position{Line: 10, Character: 5}},
			},
		}
	})

	mgr := newMockToolManager()
	mgr.clients["file:///tmp/test.go"] = client

	tool := NewLspDefinitionTool(mgr)
	params, _ := json.Marshal(map[string]interface{}{
		"uri":       "file:///tmp/test.go",
		"line":      5,
		"character": 5,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var out struct {
		Locations []Location `json:"locations"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(out.Locations) != 1 {
		t.Fatalf("locations len = %d, want 1", len(out.Locations))
	}
	if out.Locations[0].URI != "file:///tmp/other.go" {
		t.Errorf("uri = %q, want file:///tmp/other.go", out.Locations[0].URI)
	}
}

func TestLspDefinitionToolServerNotRunning(t *testing.T) {
	mgr := newMockToolManager()
	tool := NewLspDefinitionTool(mgr)
	params, _ := json.Marshal(map[string]interface{}{
		"uri":       "file:///tmp/test.go",
		"line":      0,
		"character": 0,
	})

	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Error("expected error when server not running, got nil")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// lsp_references tool
// ──────────────────────────────────────────────────────────────────────────────

func TestLspReferencesToolName(t *testing.T) {
	mgr := newMockToolManager()
	tool := NewLspReferencesTool(mgr)
	if tool.Name() != "lsp_references" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "lsp_references")
	}
}

func TestLspReferencesToolDelegatesToClient(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	server.Handle("textDocument/references", func(params json.RawMessage) interface{} {
		return []Location{
			{
				URI:   "file:///tmp/a.go",
				Range: Range{Start: Position{Line: 1, Character: 0}, End: Position{Line: 1, Character: 3}},
			},
			{
				URI:   "file:///tmp/b.go",
				Range: Range{Start: Position{Line: 5, Character: 2}, End: Position{Line: 5, Character: 5}},
			},
		}
	})

	mgr := newMockToolManager()
	mgr.clients["file:///tmp/test.go"] = client

	tool := NewLspReferencesTool(mgr)
	params, _ := json.Marshal(map[string]interface{}{
		"uri":                 "file:///tmp/test.go",
		"line":                3,
		"character":           1,
		"include_declaration": true,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var out struct {
		Locations []Location `json:"locations"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(out.Locations) != 2 {
		t.Fatalf("locations len = %d, want 2", len(out.Locations))
	}
}

func TestLspReferencesToolServerNotRunning(t *testing.T) {
	mgr := newMockToolManager()
	tool := NewLspReferencesTool(mgr)
	params, _ := json.Marshal(map[string]interface{}{
		"uri":       "file:///tmp/test.go",
		"line":      0,
		"character": 0,
	})

	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Error("expected error when server not running, got nil")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Tools() helper
// ──────────────────────────────────────────────────────────────────────────────

func TestToolsReturnsAllLspTools(t *testing.T) {
	mgr := newMockToolManager()
	tools := LspTools(mgr)

	if len(tools) != 4 {
		t.Fatalf("LspTools() len = %d, want 4", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
		if tool.Description() == "" {
			t.Errorf("tool %q has empty description", tool.Name())
		}
	}

	expected := []string{"lsp_diagnostics", "lsp_hover", "lsp_definition", "lsp_references"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing tool %q", name)
		}
	}
}
