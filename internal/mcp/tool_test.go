package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestMCPToolImplementsCoreTool(t *testing.T) {
	// Verify MCPTool implements core.Tool interface
	var _ interface {
		Name() string
		Description() string
		Schema() string
		Execute(ctx context.Context, args json.RawMessage) (string, error)
	} = &MCPTool{}
}

func TestMCPToolName(t *testing.T) {
	tool := &MCPTool{
		name:       "docs_search",
		serverName: "docs",
		toolName:   "search",
	}

	got := tool.Name()
	if got != "docs_search" {
		t.Errorf("Name() = %q, want %q", got, "docs_search")
	}
}

func TestMCPToolDescription(t *testing.T) {
	tool := &MCPTool{
		description: "Search documentation",
	}

	got := tool.Description()
	if got != "Search documentation" {
		t.Errorf("Description() = %q, want %q", got, "Search documentation")
	}
}

func TestMCPToolSchema(t *testing.T) {
	schema := `{"type":"object","properties":{"query":{"type":"string"}}}`
	tool := &MCPTool{
		inputSchema: schema,
	}

	got := tool.Schema()
	if got != schema {
		t.Errorf("Schema() = %q, want %q", got, schema)
	}
}

func TestMCPToolExecute(t *testing.T) {
	// Create a mock client that returns a successful tool call result
	cannedResult := `{"content":[{"type":"text","text":"search result"}],"isError":false}`
	client, err := newMockClient(t, cannedResult)
	if err != nil {
		t.Fatalf("newMockClient: %v", err)
	}

	tool := &MCPTool{
		name:       "docs_search",
		client:     client,
		serverName: "docs",
		toolName:   "search",
	}

	ctx := context.Background()
	args := json.RawMessage(`{"query":"test"}`)

	got, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if got != "search result" {
		t.Errorf("Execute() = %q, want %q", got, "search result")
	}
}

func TestMCPToolExecuteIsError(t *testing.T) {
	// Create a mock client that returns an error response
	cannedResult := `{"content":[{"type":"text","text":"tool failed"}],"isError":true}`
	client, err := newMockClient(t, cannedResult)
	if err != nil {
		t.Fatalf("newMockClient: %v", err)
	}

	tool := &MCPTool{
		name:       "docs_search",
		client:     client,
		serverName: "docs",
		toolName:   "search",
	}

	ctx := context.Background()
	args := json.RawMessage(`{"query":"test"}`)

	_, err = tool.Execute(ctx, args)
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}

	// Verify it's an MCPToolError
	var toolErr *MCPToolError
	if !errors.As(err, &toolErr) {
		t.Errorf("Execute() error type = %T, want *MCPToolError", err)
	}
}

func TestMCPToolExecuteMultipleContent(t *testing.T) {
	// Test multiple text content items are concatenated
	cannedResult := `{"content":[{"type":"text","text":"line1"},{"type":"text","text":"line2"}],"isError":false}`
	client, err := newMockClient(t, cannedResult)
	if err != nil {
		t.Fatalf("newMockClient: %v", err)
	}

	tool := &MCPTool{
		name:       "docs_search",
		client:     client,
		serverName: "docs",
		toolName:   "search",
	}

	ctx := context.Background()
	args := json.RawMessage(`{}`)

	got, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	expected := "line1\nline2"
	if got != expected {
		t.Errorf("Execute() = %q, want %q", got, expected)
	}
}
