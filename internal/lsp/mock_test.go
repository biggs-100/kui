package lsp

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// MockLspServer is a pipe-based mock LSP server for testing.
type MockLspServer struct {
	mu       sync.Mutex
	stdin    io.Reader
	stdout   io.Writer
	handlers map[string]func(params json.RawMessage) interface{}
	closed   bool
}

// NewMockLspServer creates a mock LSP server backed by io.Pipe pairs.
func NewMockLspServer() (*MockLspServer, *LspClient, error) {
	// Server stdin (reads from client's stdin writes)
	serverStdinR, clientStdinW := io.Pipe()
	// Server stdout (writes to client's stdout reads)
	clientStdoutR, serverStdoutW := io.Pipe()

	server := &MockLspServer{
		stdin:    serverStdinR,
		stdout:   serverStdoutW,
		handlers: make(map[string]func(params json.RawMessage) interface{}),
	}

	// Build a client that uses the pipes directly
	client := &LspClient{
		cmd:     nil,
		stdin:   &pipeWriter{clientStdinW},
		stdout:  &pipeReader{clientStdoutR},
		pending: make(map[int]chan jsonrpcResponse),
	}

	// Register default handlers
	server.HandleInitialize()

	go server.serve()
	go client.handleMessages()

	return server, client, nil
}

// HandleInitialize registers the default initialize handler.
func (s *MockLspServer) HandleInitialize() {
	s.Handle("initialize", func(params json.RawMessage) interface{} {
		return map[string]interface{}{
			"capabilities": map[string]interface{}{
				"textDocumentSync": map[string]interface{}{
					"openClose": true,
					"change":    1,
				},
				"hoverProvider":      true,
				"definitionProvider": true,
				"referencesProvider": true,
				"publishDiagnostics": true,
			},
			"serverInfo": map[string]string{
				"name":    "mock-lsp",
				"version": "0.1.0",
			},
		}
	})
}

// Handle registers a handler for a given method.
func (s *MockLspServer) Handle(method string, handler func(params json.RawMessage) interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

// serve reads requests and dispatches to handlers.
func (s *MockLspServer) serve() {
	for {
		data, err := ReadMessage(s.stdin)
		if err != nil {
			return
		}

		var req struct {
			ID     *int            `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			continue
		}

		// Notification (no ID) — handle but don't respond.
		if req.ID == nil {
			if req.Method == "initialized" || req.Method == "exit" || req.Method == "shutdown" {
				continue
			}
			s.mu.Lock()
			handler, ok := s.handlers[req.Method]
			s.mu.Unlock()
			if ok {
				handler(req.Params)
			}
			continue
		}

		s.mu.Lock()
		handler, ok := s.handlers[req.Method]
		s.mu.Unlock()

		var resp jsonrpcResponse
		resp.JSONRPC = "2.0"
		resp.ID = req.ID

		if !ok {
			errCode := -32601 // Method not found
			resp.Error = &jsonrpcError{Code: errCode, Message: fmt.Sprintf("method not found: %s", req.Method)}
		} else {
			result := handler(req.Params)
			raw, _ := json.Marshal(result)
			resp.Result = raw
		}

		respData, _ := json.Marshal(resp)
		WriteMessage(s.stdout, respData)
	}
}

// Close shuts down the mock server.
func (s *MockLspServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	if w, ok := s.stdout.(io.Closer); ok {
		w.Close()
	}
	if r, ok := s.stdin.(io.Closer); ok {
		r.Close()
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// IO adapters for pipe-based mock
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
