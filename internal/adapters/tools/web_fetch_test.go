package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ── Task 3.3 RED: TestWebFetchValidURL ──────────────────────────────────────

func TestWebFetchValidURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><head><title>Test Page</title></head><body>Hello World</body></html>"))
	}))
	defer srv.Close()

	tool := NewWebFetchTool(0)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+srv.URL+`"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if raw == "" {
		t.Fatal("expected non-empty body")
	}
	if !strings.Contains(raw, "Hello World") {
		t.Errorf("body does not contain expected content, got: %q", raw)
	}
}

// ── Task 3.5 RED: TestWebFetchInvalidURL ────────────────────────────────────

func TestWebFetchInvalidURL(t *testing.T) {
	tool := NewWebFetchTool(0)
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"ftp://example.com/file"}`))
	if err == nil {
		t.Fatal("expected error for ftp:// URL, got nil")
	}
}

// ── Task 3.6 RED: TestWebFetchNetworkError ──────────────────────────────────

func TestWebFetchNetworkError(t *testing.T) {
	tool := NewWebFetchTool(0)
	// Use a port that is almost certainly not listening.
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"http://127.0.0.1:1"}`))
	if err == nil {
		t.Fatal("expected network error for unreachable host, got nil")
	}
}

// ── Task 3.7 RED: TestWebFetchTimeout ───────────────────────────────────────

func TestWebFetchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("delayed"))
	}))
	defer srv.Close()

	// Use a very short timeout so the test completes quickly.
	tool := NewWebFetchTool(100 * time.Millisecond)
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+srv.URL+`"}`))
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// ── Format tests: web_fetch format parameter ───────────────────────────────

const testHTMLBody = `<html><head><title>Test</title></head><body><h1>Hello</h1><p>  World  </p></body></html>`

func TestWebFetchFormatText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testHTMLBody))
	}))
	defer srv.Close()

	tool := NewWebFetchTool(0)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+srv.URL+`","format":"text"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if raw != testHTMLBody {
		t.Errorf("format=text should return raw body exactly; got %q", raw)
	}
}

func TestWebFetchFormatHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testHTMLBody))
	}))
	defer srv.Close()

	tool := NewWebFetchTool(0)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+srv.URL+`","format":"html"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if raw != testHTMLBody {
		t.Errorf("format=html should return raw HTML exactly; got %q", raw)
	}
}

func TestWebFetchFormatMarkdown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testHTMLBody))
	}))
	defer srv.Close()

	tool := NewWebFetchTool(0)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+srv.URL+`","format":"markdown"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if strings.Contains(raw, "<") || strings.Contains(raw, ">") {
		t.Errorf("format=markdown should strip HTML tags, got: %q", raw)
	}
	if !strings.Contains(raw, "Hello") || !strings.Contains(raw, "World") {
		t.Errorf("format=markdown should contain text content, got: %q", raw)
	}
}

func TestWebFetchFormatDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testHTMLBody))
	}))
	defer srv.Close()

	tool := NewWebFetchTool(0)
	// No format field — should default to "text" (raw body)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+srv.URL+`"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if raw != testHTMLBody {
		t.Errorf("default format should return raw body exactly; got %q", raw)
	}
}
