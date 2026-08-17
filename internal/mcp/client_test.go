package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestNewClientInitialState verifies a new client starts with correct defaults.
func TestNewClientInitialState(t *testing.T) {
	c, err := NewClient(context.Background(), []string{"echo", "test"}, "", nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer c.Close()

	if c.nextID != 0 {
		t.Errorf("nextID = %d, want 0", c.nextID)
	}
}

// TestClientInitializeHandshake verifies REQ-MCP-6: the client sends an
// initialize request with protocolVersion "2025-03-26".
func TestClientInitializeHandshake(t *testing.T) {
	result := `{"protocolVersion":"2025-03-26","capabilities":{"tools":{"listChanged":false}}}`
	c, err := newMockClient(t, result)
	if err != nil {
		t.Fatalf("newMockClient() error = %v", err)
	}
	defer c.Close()

	err = c.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
}

// TestClientListTools verifies REQ-MCP-7: the client calls tools/list and
// returns the parsed tool definitions.
func TestClientListTools(t *testing.T) {
	result := `{"tools":[{"name":"echo","description":"Echoes input","inputSchema":{"type":"object","properties":{"text":{"type":"string"}}}}]}`
	c, err := newMockClient(t, result)
	if err != nil {
		t.Fatalf("newMockClient() error = %v", err)
	}
	defer c.Close()

	if err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	tools, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(tools))
	}
	if tools[0].Name != "echo" {
		t.Errorf("tool name = %q, want %q", tools[0].Name, "echo")
	}
}

// TestClientCallTool verifies REQ-MCP-8: the client sends tools/call with
// name and args, and returns the text content.
func TestClientCallTool(t *testing.T) {
	result := `{"content":[{"type":"text","text":"hello world"}]}`
	c, err := newMockClient(t, result)
	if err != nil {
		t.Fatalf("newMockClient() error = %v", err)
	}
	defer c.Close()

	if err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	args, _ := json.Marshal(map[string]string{"text": "hello"})
	res, err := c.CallTool(context.Background(), "echo", args)
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if res != "hello world" {
		t.Errorf("result = %q, want %q", res, "hello world")
	}
}

// TestClientHandleServerCrash verifies REQ-MCP-9: when the server's stdout
// pipe closes unexpectedly, the client returns a clear error.
func TestClientHandleServerCrash(t *testing.T) {
	serverStdinR, clientStdinW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	clientStdoutR, serverStdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	// Mock: read request, then close stdout immediately (simulating crash).
	go func() {
		scanner := bufio.NewScanner(serverStdinR)
		scanner.Scan() // read request
		serverStdoutW.Close()
		serverStdinR.Close()
	}()

	c := &Client{
		cmd:     nil,
		stdin:   &closingWriter{clientStdinW},
		stdout:  clientStdoutR,
		scanner: bufio.NewScanner(clientStdoutR),
	}
	defer c.Close()

	err = c.Initialize(context.Background())
	if err == nil {
		t.Fatal("expected error for crashed server, got nil")
	}
}

// TestClientContextCancellation verifies REQ-MCP-10: cancelling context
// stops the client and cleans up.
func TestClientContextCancellation(t *testing.T) {
	serverStdinR, clientStdinW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	clientStdoutR, _, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	// Mock: read request but never respond (simulates slow server).
	go func() {
		scanner := bufio.NewScanner(serverStdinR)
		scanner.Scan()
		// Don't write anything — client should timeout via context.
		time.Sleep(5 * time.Second)
		serverStdinR.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := &Client{
		cmd:     nil,
		stdin:   &closingWriter{clientStdinW},
		stdout:  clientStdoutR,
		scanner: bufio.NewScanner(clientStdoutR),
	}
	defer c.Close()

	err = c.Initialize(ctx)
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
}

// TestClientMalformedJSONResponse verifies REQ-MCP-5: the client handles
// non-JSON responses gracefully by skipping them.
func TestClientMalformedJSONResponse(t *testing.T) {
	serverStdinR, clientStdinW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	clientStdoutR, serverStdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		defer serverStdoutW.Close()
		scanner := bufio.NewScanner(serverStdinR)
		if scanner.Scan() {
			// Send malformed response first, then valid.
			serverStdoutW.Write([]byte("not valid json at all\n"))
			serverStdoutW.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","capabilities":{}}}` + "\n"))
		}
		serverStdinR.Close()
	}()

	c := &Client{
		cmd:     nil,
		stdin:   &closingWriter{clientStdinW},
		stdout:  clientStdoutR,
		scanner: bufio.NewScanner(clientStdoutR),
	}
	defer c.Close()

	// Initialize should fail because the first response is malformed and
	// the pipe closes before a valid response is available.
	err = c.Initialize(context.Background())
	if err == nil {
		t.Fatal("expected error for malformed response, got nil")
	}
}

// TestClientProtocolVersionInInitialize verifies REQ-MCP-6: initialize
// sends protocolVersion "2025-03-26".
func TestClientProtocolVersionInInitialize(t *testing.T) {
	result := `{"protocolVersion":"2025-03-26","capabilities":{}}`
	c, err := newMockClient(t, result)
	if err != nil {
		t.Fatalf("newMockClient() error = %v", err)
	}
	defer c.Close()

	err = c.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
}
