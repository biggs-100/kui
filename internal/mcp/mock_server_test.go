package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// Pipe-based mock client for client_test.go compatibility
// ──────────────────────────────────────────────────────────────────────────────

// newMockClient creates a Client backed by io.Pipe pairs for testing.
func newMockClient(t *testing.T, cannedResult string) (*Client, error) {
	t.Helper()

	r, w := io.Pipe()
	clientR, serverW := io.Pipe()

	server := &mockServer{
		stdin:        r,
		stdout:       serverW,
		cannedResult: cannedResult,
	}
	go server.run()

	return &Client{
		cmd:     nil,
		stdin:   &pipeWriter{w},
		stdout:  &pipeReader{clientR},
		scanner: bufio.NewScanner(clientR),
	}, nil
}

type mockServer struct {
	stdin        io.Reader
	stdout       io.Writer
	cannedResult string
}

func (s *mockServer) run() {
	scanner := bufio.NewScanner(s.stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		if req.ID == 0 {
			continue
		}

		var resp string
		switch {
		case req.Method == "initialize":
			initResult := `{"protocolVersion":"2025-03-26","capabilities":{},"serverInfo":{"name":"mock","version":"0.1.0"}}`
			resp = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":%s}`, req.ID, initResult)
		default:
			resp = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":%s}`, req.ID, s.cannedResult)
		}

		fmt.Fprintf(s.stdout, "%s\n", resp)
	}
}

// newMockClientWithTools creates a mock client with specific tool definitions.
func newMockClientWithTools(tools []MCPToolDef) (*Client, error) {
	r, w := io.Pipe()
	clientR, serverW := io.Pipe()

	server := &mockServerWithTools{
		stdin:  r,
		stdout: serverW,
		tools:  tools,
	}
	go server.run()

	return &Client{
		cmd:     nil,
		stdin:   &pipeWriter{w},
		stdout:  &pipeReader{clientR},
		scanner: bufio.NewScanner(clientR),
	}, nil
}

type mockServerWithTools struct {
	stdin  io.Reader
	stdout io.Writer
	tools  []MCPToolDef
}

func (s *mockServerWithTools) run() {
	scanner := bufio.NewScanner(s.stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		if req.ID == 0 {
			continue
		}

		var resp string
		switch req.Method {
		case "initialize":
			initResult := `{"protocolVersion":"2025-03-26","capabilities":{},"serverInfo":{"name":"mock","version":"0.1.0"}}`
			resp = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":%s}`, req.ID, initResult)
		case "tools/list":
			toolsJSON, _ := json.Marshal(s.tools)
			resp = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":{"tools":%s}}`, req.ID, toolsJSON)
		case "tools/call":
			result := `{"content":[{"type":"text","text":"tool result"}],"isError":false}`
			resp = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":%s}`, req.ID, result)
		default:
			resp = fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":{}}`, req.ID)
		}

		fmt.Fprintf(s.stdout, "%s\n", resp)
	}
}

// mockClientFactory creates a Client factory that returns mock clients.
func mockClientFactory(cannedTools []MCPToolDef) ClientFactory {
	return func(ctx context.Context, name string, cfg ServerConfig) (*Client, error) {
		return newMockClientWithTools(cannedTools)
	}
}

// mockManagerWithTools creates a manager with a factory for each server.
func mockManagerWithTools(t *testing.T, servers map[string]ServerConfig, serverTools map[string][]MCPToolDef) *MCPManager {
	t.Helper()

	factory := func(ctx context.Context, name string, cfg ServerConfig) (*Client, error) {
		tools, ok := serverTools[name]
		if !ok {
			tools = []MCPToolDef{}
		}
		return newMockClientWithTools(tools)
	}

	cfg := &Config{Servers: servers}
	return NewMCPManagerWithFactory(cfg, factory)
}

// ──────────────────────────────────────────────────────────────────────────────
// IO adapters
// ──────────────────────────────────────────────────────────────────────────────

type pipeWriter struct {
	pw *io.PipeWriter
}

func (w *pipeWriter) Write(p []byte) (n int, err error) {
	return w.pw.Write(p)
}

func (w *pipeWriter) Close() error {
	return w.pw.Close()
}

type pipeReader struct {
	pr *io.PipeReader
}

func (r *pipeReader) Read(p []byte) (n int, err error) {
	return r.pr.Read(p)
}

func (r *pipeReader) Close() error {
	return r.pr.Close()
}

// closingWriter wraps *os.File to satisfy io.WriteCloser (used by client_test.go).
type closingWriter struct {
	f *os.File
}

func (w *closingWriter) Write(p []byte) (n int, err error) {
	return w.f.Write(p)
}

func (w *closingWriter) Close() error {
	return w.f.Close()
}
