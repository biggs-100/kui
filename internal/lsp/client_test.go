package lsp

import (
	"encoding/json"
	"testing"
)

func TestNewLspClientEmptyPath(t *testing.T) {
	_, err := NewLspClient("", nil, "")
	if err == nil {
		t.Fatal("expected error for empty server path, got nil")
	}
}

func TestNewLspClientValid(t *testing.T) {
	c, err := NewLspClient("echo", []string{}, "file:///tmp")
	if err != nil {
		t.Fatalf("NewLspClient() error: %v", err)
	}
	defer c.Stop()

	if c.pending == nil {
		t.Error("pending map should be initialized")
	}
}

func TestMockServerInitialize(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	err = client.Initialize("file:///tmp/project")
	if err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}
}

func TestMockServerSendRequest(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	// Register a custom handler
	server.Handle("textDocument/hover", func(params json.RawMessage) interface{} {
		return Hover{
			Contents: "func foo() string",
		}
	})

	// Initialize first
	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	// Send hover request
	var result Hover
	err = client.SendRequest("textDocument/hover", map[string]interface{}{
		"textDocument": TextDocumentIdentifier{URI: "file:///tmp/test.go"},
		"position":     Position{Line: 5, Character: 2},
	}, &result)
	if err != nil {
		t.Fatalf("SendRequest(hover) error: %v", err)
	}

	if result.Contents != "func foo() string" {
		t.Errorf("hover contents = %q, want %q", result.Contents, "func foo() string")
	}
}

func TestMockServerSendNotification(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	// Send a notification (no response expected)
	err = client.SendNotification("textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        "file:///tmp/test.go",
			"languageId": "go",
			"version":    1,
			"text":       "package main",
		},
	})
	if err != nil {
		t.Fatalf("SendNotification() error: %v", err)
	}
}

func TestMockServerMethodNotFound(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	// Send request to unregistered method
	var result json.RawMessage
	err = client.SendRequest("unknown/method", nil, &result)
	if err == nil {
		t.Fatal("expected error for unknown method, got nil")
	}

	lspErr, ok := err.(*LspError)
	if !ok {
		t.Fatalf("expected *LspError, got %T", err)
	}
	if lspErr.Code != -32601 {
		t.Errorf("error code = %d, want -32601", lspErr.Code)
	}
}

func TestMockServerStop(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	err = client.Stop()
	if err != nil {
		t.Fatalf("Stop() error: %v", err)
	}

	// After stop, requests should fail
	var result json.RawMessage
	err = client.SendRequest("initialize", nil, &result)
	if err == nil {
		t.Fatal("expected error after Stop, got nil")
	}
}

func TestMockServerNotificationHandler(t *testing.T) {
	server, client, err := NewMockLspServer()
	if err != nil {
		t.Fatalf("NewMockLspServer() error: %v", err)
	}
	defer server.Close()
	defer client.Stop()

	// Set up notification handler
	received := make(chan string, 1)
	client.SetNotificationHandler(func(method string, params json.RawMessage) {
		received <- method
	})

	if err := client.Initialize("file:///tmp/project"); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	// Have the server send a notification
	server.Handle("textDocument/publishDiagnostics", func(params json.RawMessage) interface{} {
		return nil
	})

	// Trigger the server to send a notification by calling our custom handler via request
	// Since we can't push notifications from the mock server, we test the handler registration
	// by verifying the client's handler was set.
	client.mu.Lock()
	handler := client.onNotification
	client.mu.Unlock()

	if handler == nil {
		t.Error("notification handler should be set")
	}
}
