package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// newMockClient creates a Client backed by controlled os.Pipe pairs.
// A goroutine acts as the mock MCP server: it reads JSON-RPC requests from
// stdin, echoes the request ID, and writes the canned result payload.
// The result is wrapped in a proper JSON-RPC 2.0 response envelope.
func newMockClient(t *testing.T, cannedResult string) (*Client, error) {
	t.Helper()

	serverStdinR, clientStdinW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	clientStdoutR, serverStdoutW, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	go mockServerLoop(serverStdinR, serverStdoutW, cannedResult)

	return &Client{
		cmd:     nil,
		stdin:   &closingWriter{clientStdinW},
		stdout:  clientStdoutR,
		scanner: bufio.NewScanner(clientStdoutR),
	}, nil
}

// mockServerLoop reads JSON-RPC requests and responds with the canned result,
// echoing the request ID. It continues until the input pipe closes.
func mockServerLoop(stdin *os.File, stdout *os.File, cannedResult string) {
	defer stdout.Close()
	defer stdin.Close()

	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse just the ID from the request.
		var req struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		// Skip notifications (no ID).
		if req.ID == 0 {
			continue
		}

		// Build a proper JSON-RPC 2.0 response with matching ID.
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":%s}`, req.ID, cannedResult)
		stdout.Write([]byte(resp + "\n"))
	}
}

// closingWriter wraps *os.File to satisfy io.WriteCloser.
type closingWriter struct {
	f *os.File
}

func (w *closingWriter) Write(p []byte) (n int, err error) {
	return w.f.Write(p)
}

func (w *closingWriter) Close() error {
	return w.f.Close()
}
