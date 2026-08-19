package lsp

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// ──────────────────────────────────────────────────────────────────────────────
// DidOpen
// ──────────────────────────────────────────────────────────────────────────────

func TestDidOpen(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	// Track received notifications with synchronization
	var mu sync.Mutex
	var receivedMethod string
	var receivedParams map[string]interface{}
	done := make(chan struct{})

	server.Handle("textDocument/didOpen", func(params json.RawMessage) interface{} {
		var p map[string]interface{}
		json.Unmarshal(params, &p)
		mu.Lock()
		receivedMethod = "textDocument/didOpen"
		receivedParams = p
		mu.Unlock()
		close(done)
		return nil
	})

	err = client.DidOpen("file:///tmp/test.go", "go", 1, "package main")
	if err != nil {
		t.Fatalf("DidOpen() error: %v", err)
	}

	// Wait for the handler to be called
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for didOpen notification")
	}

	mu.Lock()
	defer mu.Unlock()
	if receivedMethod != "textDocument/didOpen" {
		t.Errorf("method = %q, want %q", receivedMethod, "textDocument/didOpen")
	}

	td, ok := receivedParams["textDocument"].(map[string]interface{})
	if !ok {
		t.Fatal("missing textDocument in params")
	}
	if td["uri"] != "file:///tmp/test.go" {
		t.Errorf("uri = %v, want file:///tmp/test.go", td["uri"])
	}
	if td["languageId"] != "go" {
		t.Errorf("languageId = %v, want go", td["languageId"])
	}
	// JSON numbers are float64
	if td["version"].(float64) != 1 {
		t.Errorf("version = %v, want 1", td["version"])
	}
	if td["text"] != "package main" {
		t.Errorf("text = %v, want %q", td["text"], "package main")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// DidClose
// ──────────────────────────────────────────────────────────────────────────────

func TestDidClose(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	var mu sync.Mutex
	var receivedURI string
	done := make(chan struct{})

	server.Handle("textDocument/didClose", func(params json.RawMessage) interface{} {
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		json.Unmarshal(params, &p)
		mu.Lock()
		receivedURI = p.TextDocument.URI
		mu.Unlock()
		close(done)
		return nil
	})

	err = client.DidClose("file:///tmp/test.go")
	if err != nil {
		t.Fatalf("DidClose() error: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for didClose notification")
	}

	mu.Lock()
	defer mu.Unlock()
	if receivedURI != "file:///tmp/test.go" {
		t.Errorf("uri = %q, want %q", receivedURI, "file:///tmp/test.go")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// DidChange
// ──────────────────────────────────────────────────────────────────────────────

func TestDidChange(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	var mu sync.Mutex
	var receivedURI string
	var receivedVersion int
	var receivedChanges []TextDocumentContentChangeEvent
	done := make(chan struct{})

	server.Handle("textDocument/didChange", func(params json.RawMessage) interface{} {
		var p struct {
			TextDocument struct {
				URI     string `json:"uri"`
				Version int    `json:"version"`
			} `json:"textDocument"`
			ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
		}
		json.Unmarshal(params, &p)
		mu.Lock()
		receivedURI = p.TextDocument.URI
		receivedVersion = p.TextDocument.Version
		receivedChanges = p.ContentChanges
		mu.Unlock()
		close(done)
		return nil
	})

	changes := []TextDocumentContentChangeEvent{
		{Text: "package main\n\nfunc main() {}"},
	}
	err = client.DidChange("file:///tmp/test.go", 2, changes)
	if err != nil {
		t.Fatalf("DidChange() error: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for didChange notification")
	}

	mu.Lock()
	defer mu.Unlock()
	if receivedURI != "file:///tmp/test.go" {
		t.Errorf("uri = %q, want file:///tmp/test.go", receivedURI)
	}
	if receivedVersion != 2 {
		t.Errorf("version = %d, want 2", receivedVersion)
	}
	if len(receivedChanges) != 1 {
		t.Fatalf("contentChanges len = %d, want 1", len(receivedChanges))
	}
	if receivedChanges[0].Text != "package main\n\nfunc main() {}" {
		t.Errorf("text = %q, want %q", receivedChanges[0].Text, "package main\n\nfunc main() {}")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Hover
// ──────────────────────────────────────────────────────────────────────────────

func TestHover(t *testing.T) {
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
		return Hover{
			Contents: "func main()",
			Range: &Range{
				Start: Position{Line: 5, Character: 5},
				End:   Position{Line: 5, Character: 9},
			},
		}
	})

	result, err := client.Hover("file:///tmp/test.go", 5, 5)
	if err != nil {
		t.Fatalf("Hover() error: %v", err)
	}
	if result.Contents != "func main()" {
		t.Errorf("contents = %q, want %q", result.Contents, "func main()")
	}
	if result.Range == nil {
		t.Fatal("range should not be nil")
	}
	if result.Range.Start.Line != 5 || result.Range.Start.Character != 5 {
		t.Errorf("range.start = %+v, want {Line:5 Character:5}", result.Range.Start)
	}
}

func TestHoverNotFound(t *testing.T) {
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
		return nil // null result = no hover info
	})

	result, err := client.Hover("file:///tmp/test.go", 0, 0)
	if err != nil {
		t.Fatalf("Hover() error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil hover, got %+v", result)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Definition
// ──────────────────────────────────────────────────────────────────────────────

func TestDefinition(t *testing.T) {
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
				URI: "file:///tmp/other.go",
				Range: Range{
					Start: Position{Line: 10, Character: 0},
					End:   Position{Line: 10, Character: 5},
				},
			},
		}
	})

	locations, err := client.Definition("file:///tmp/test.go", 5, 5)
	if err != nil {
		t.Fatalf("Definition() error: %v", err)
	}
	if len(locations) != 1 {
		t.Fatalf("locations len = %d, want 1", len(locations))
	}
	if locations[0].URI != "file:///tmp/other.go" {
		t.Errorf("uri = %q, want file:///tmp/other.go", locations[0].URI)
	}
	if locations[0].Range.Start.Line != 10 {
		t.Errorf("range.start.line = %d, want 10", locations[0].Range.Start.Line)
	}
}

func TestDefinitionNotFound(t *testing.T) {
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
		return nil // no definition found
	})

	locations, err := client.Definition("file:///tmp/test.go", 0, 0)
	if err != nil {
		t.Fatalf("Definition() error: %v", err)
	}
	if len(locations) != 0 {
		t.Errorf("expected empty locations, got %d", len(locations))
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// References
// ──────────────────────────────────────────────────────────────────────────────

func TestReferences(t *testing.T) {
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
				URI: "file:///tmp/a.go",
				Range: Range{
					Start: Position{Line: 1, Character: 0},
					End:   Position{Line: 1, Character: 3},
				},
			},
			{
				URI: "file:///tmp/b.go",
				Range: Range{
					Start: Position{Line: 5, Character: 2},
					End:   Position{Line: 5, Character: 5},
				},
			},
		}
	})

	locations, err := client.References("file:///tmp/test.go", 3, 1, true)
	if err != nil {
		t.Fatalf("References() error: %v", err)
	}
	if len(locations) != 2 {
		t.Fatalf("locations len = %d, want 2", len(locations))
	}
	if locations[0].URI != "file:///tmp/a.go" {
		t.Errorf("locations[0].uri = %q, want file:///tmp/a.go", locations[0].URI)
	}
	if locations[1].URI != "file:///tmp/b.go" {
		t.Errorf("locations[1].uri = %q, want file:///tmp/b.go", locations[1].URI)
	}
}

func TestReferencesNone(t *testing.T) {
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
		return []Location{} // empty list
	})

	locations, err := client.References("file:///tmp/test.go", 0, 0, false)
	if err != nil {
		t.Fatalf("References() error: %v", err)
	}
	if len(locations) != 0 {
		t.Errorf("expected empty locations, got %d", len(locations))
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Diagnostics (pull-based)
// ──────────────────────────────────────────────────────────────────────────────

func TestDiagnostics(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	server.Handle("textDocument/diagnostic", func(params json.RawMessage) interface{} {
		return []Diagnostic{
			{
				Range: Range{
					Start: Position{Line: 3, Character: 0},
					End:   Position{Line: 3, Character: 10},
				},
				Severity: DiagnosticSeverityError,
				Message:  "undefined: foo",
				Source:   "gopls",
			},
		}
	})

	diags, err := client.Diagnostics("file:///tmp/test.go")
	if err != nil {
		t.Fatalf("Diagnostics() error: %v", err)
	}
	if len(diags) != 1 {
		t.Fatalf("diagnostics len = %d, want 1", len(diags))
	}
	if diags[0].Severity != DiagnosticSeverityError {
		t.Errorf("severity = %d, want %d", diags[0].Severity, DiagnosticSeverityError)
	}
	if diags[0].Message != "undefined: foo" {
		t.Errorf("message = %q, want %q", diags[0].Message, "undefined: foo")
	}
}

func TestDiagnosticsEmpty(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	server.Handle("textDocument/diagnostic", func(params json.RawMessage) interface{} {
		return []Diagnostic{}
	})

	diags, err := client.Diagnostics("file:///tmp/test.go")
	if err != nil {
		t.Fatalf("Diagnostics() error: %v", err)
	}
	if len(diags) != 0 {
		t.Errorf("expected empty diagnostics, got %d", len(diags))
	}
}
