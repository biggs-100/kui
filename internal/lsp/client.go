package lsp

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

// jsonrpcRequest is a JSON-RPC 2.0 request frame.
type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonrpcResponse is a JSON-RPC 2.0 response frame.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcError is the error object inside a JSON-RPC 2.0 response.
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// LspClient communicates with an LSP server subprocess via JSON-RPC 2.0
// over stdio with Content-Length framing.
type LspClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	mu        sync.Mutex
	pending   map[int]chan jsonrpcResponse
	nextID    int64
	closed    bool
	closeOnce sync.Once

	// Notification handler — called for server-initiated notifications.
	onNotification func(method string, params json.RawMessage)
}

// NewLspClient creates a new LSP client. It does NOT start the server;
// call Start() to spawn the subprocess.
func NewLspClient(serverPath string, args []string, rootUri string) (*LspClient, error) {
	if serverPath == "" {
		return nil, fmt.Errorf("lsp client: server path is required")
	}

	cmdArgs := make([]string, 0, len(args)+1)
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(serverPath, cmdArgs...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp client: stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp client: stdout pipe: %w", err)
	}

	return &LspClient{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		pending: make(map[int]chan jsonrpcResponse),
	}, nil
}

// Start spawns the LSP server process and begins reading responses.
func (c *LspClient) Start() error {
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("lsp client: start: %w", err)
	}

	go c.handleMessages()
	return nil
}

// Stop sends shutdown, exits, and kills the process.
func (c *LspClient) Stop() error {
	var err error
	c.closeOnce.Do(func() {
		// Send shutdown request (best-effort).
		c.mu.Lock()
		if !c.closed {
			_ = c.sendRaw(jsonrpcRequest{
				JSONRPC: "2.0",
				Method:  "shutdown",
			})
		}
		c.mu.Unlock()

		// Send exit notification.
		_ = c.sendRaw(jsonrpcRequest{
			JSONRPC: "2.0",
			Method:  "exit",
		})

		// Kill process.
		if c.cmd != nil && c.cmd.Process != nil {
			c.cmd.Process.Kill()
			err = c.cmd.Wait()
		}

		// Mark closed.
		c.mu.Lock()
		c.closed = true
		for id, ch := range c.pending {
			close(ch)
			delete(c.pending, id)
		}
		c.mu.Unlock()

		if c.stdin != nil {
			c.stdin.Close()
		}
		if c.stdout != nil {
			c.stdout.Close()
		}
	})
	return err
}

// Initialize sends the LSP initialize request and waits for a response.
func (c *LspClient) Initialize(rootUri string) error {
	params := map[string]interface{}{
		"processId": nil,
		"rootUri":   rootUri,
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"hover":              map[string]interface{}{"contentFormat": []string{"plaintext", "markdown"}},
				"definition":         map[string]interface{}{},
				"references":         map[string]interface{}{},
				"publishDiagnostics": map[string]interface{}{},
			},
		},
		"clientInfo": map[string]string{
			"name":    "kui",
			"version": "0.1.0",
		},
	}

	var result json.RawMessage
	if err := c.SendRequest("initialize", params, &result); err != nil {
		return fmt.Errorf("lsp initialize: %w", err)
	}

	// Send initialized notification.
	if err := c.SendNotification("initialized", nil); err != nil {
		return fmt.Errorf("lsp initialized notification: %w", err)
	}

	return nil
}

// SendRequest sends a JSON-RPC 2.0 request and waits for the matching response.
func (c *LspClient) SendRequest(method string, params interface{}, result interface{}) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client is closed")
	}

	id := int(atomic.AddInt64(&c.nextID, 1))
	ch := make(chan jsonrpcResponse, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return fmt.Errorf("marshal request: %w", err)
	}

	if err := WriteMessage(c.stdin, data); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return fmt.Errorf("write request: %w", err)
	}

	// Wait for response.
	resp, ok := <-ch
	if !ok {
		return fmt.Errorf("client closed while waiting for response")
	}

	if resp.Error != nil {
		return &LspError{Code: resp.Error.Code, Message: resp.Error.Message}
	}

	if result != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}

	return nil
}

// SendNotification sends a JSON-RPC 2.0 notification (no response expected).
func (c *LspClient) SendNotification(method string, params interface{}) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client is closed")
	}
	c.mu.Unlock()

	notif := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	return WriteMessage(c.stdin, data)
}

// SetNotificationHandler sets the callback for server-initiated notifications.
func (c *LspClient) SetNotificationHandler(handler func(method string, params json.RawMessage)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onNotification = handler
}

// handleMessages reads Content-Length framed messages from stdout and
// dispatches responses to pending request channels.
func (c *LspClient) handleMessages() {
	for {
		data, err := ReadMessage(c.stdout)
		if err != nil {
			// Server closed or read error — close all pending.
			c.mu.Lock()
			for id, ch := range c.pending {
				close(ch)
				delete(c.pending, id)
			}
			c.mu.Unlock()
			return
		}

		var msg jsonrpcResponse
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		// Notification (no ID) — dispatch to handler.
		if msg.ID == nil {
			c.mu.Lock()
			handler := c.onNotification
			c.mu.Unlock()
			if handler != nil && msg.Method != "" {
				handler(msg.Method, msg.Params)
			}
			continue
		}

		// Response — route to pending channel.
		id := *msg.ID
		c.mu.Lock()
		ch, ok := c.pending[id]
		if ok {
			delete(c.pending, id)
		}
		c.mu.Unlock()

		if ok {
			ch <- msg
		}
	}
}

// sendRaw marshals and writes a request directly to stdin.
func (c *LspClient) sendRaw(req jsonrpcRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return WriteMessage(c.stdin, data)
}

// ──────────────────────────────────────────────────────────────────────────────
// File sync methods
// ──────────────────────────────────────────────────────────────────────────────

// DidOpen sends a textDocument/didOpen notification.
func (c *LspClient) DidOpen(uri string, languageId string, version int, text string) error {
	return c.SendNotification("textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": languageId,
			"version":    version,
			"text":       text,
		},
	})
}

// DidChange sends a textDocument/didChange notification.
func (c *LspClient) DidChange(uri string, version int, changes []TextDocumentContentChangeEvent) error {
	return c.SendNotification("textDocument/didChange", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":     uri,
			"version": version,
		},
		"contentChanges": changes,
	})
}

// DidClose sends a textDocument/didClose notification.
func (c *LspClient) DidClose(uri string) error {
	return c.SendNotification("textDocument/didClose", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// LSP request methods
// ──────────────────────────────────────────────────────────────────────────────

// Hover sends a textDocument/hover request.
func (c *LspClient) Hover(uri string, line, character int) (*Hover, error) {
	var result Hover
	if err := c.SendRequest("textDocument/hover", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: character},
	}, &result); err != nil {
		return nil, fmt.Errorf("hover: %w", err)
	}
	// A null response means no hover info at this position.
	if result.Contents == "" && result.Range == nil {
		return nil, nil
	}
	return &result, nil
}

// Definition sends a textDocument/definition request.
func (c *LspClient) Definition(uri string, line, character int) ([]Location, error) {
	var result []Location
	if err := c.SendRequest("textDocument/definition", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: character},
	}, &result); err != nil {
		return nil, fmt.Errorf("definition: %w", err)
	}
	return result, nil
}

// References sends a textDocument/references request.
func (c *LspClient) References(uri string, line, character int, includeDeclaration bool) ([]Location, error) {
	var result []Location
	if err := c.SendRequest("textDocument/references", ReferenceParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: character},
		Context:      ReferenceContext{IncludeDeclaration: includeDeclaration},
	}, &result); err != nil {
		return nil, fmt.Errorf("references: %w", err)
	}
	return result, nil
}

// Diagnostics sends a textDocument/diagnostic request (pull-based) for initial sync.
func (c *LspClient) Diagnostics(uri string) ([]Diagnostic, error) {
	var result []Diagnostic
	if err := c.SendRequest("textDocument/diagnostic", map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri},
	}, &result); err != nil {
		return nil, fmt.Errorf("diagnostics: %w", err)
	}
	return result, nil
}
