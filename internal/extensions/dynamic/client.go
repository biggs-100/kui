package dynamic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

const protocolVersion = "kui-ext/1"

// ToolDef describes a tool returned by a dynamic extension's extensions/list response.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// jsonrpcRequest is a JSON-RPC 2.0 request frame.
type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonrpcResponse is a JSON-RPC 2.0 response frame.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

// jsonrpcError is the error object inside a JSON-RPC 2.0 response.
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// listToolsResult is the shape of the extensions/list result.
type listToolsResult struct {
	Tools      []ToolDef `json:"tools"`
	NextCursor string    `json:"nextCursor,omitempty"`
}

// callToolResult is the shape of the extensions/call result.
type callToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError,omitempty"`
}

// Client communicates with a dynamic extension subprocess via JSON-RPC 2.0 over stdio.
type Client struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	scanner   *bufio.Scanner
	mu        sync.Mutex
	nextID    int
	closed    bool
	closeOnce sync.Once
}

// NewClient spawns the extension subprocess and prepares the stdio transport.
// entryPoint is the path to the extension executable.
func NewClient(ctx context.Context, entryPoint string) (*Client, error) {
	if entryPoint == "" {
		return nil, fmt.Errorf("dynamic client: entry point is required")
	}

	cmd := exec.CommandContext(ctx, entryPoint)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("dynamic client: stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("dynamic client: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("dynamic client: start: %w", err)
	}

	return &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		scanner: bufio.NewScanner(stdout),
	}, nil
}

// Initialize sends the extension initialize handshake. It must be
// called before any other requests.
func (c *Client) Initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "kui",
			"version": "0.1.0",
		},
	}

	var result json.RawMessage
	if err := c.sendRequest(ctx, "initialize", params, &result); err != nil {
		return fmt.Errorf("dynamic initialize: %w", err)
	}

	// Verify protocol version in response.
	var initResult struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(result, &initResult); err != nil {
		return fmt.Errorf("dynamic initialize: parse result: %w", err)
	}
	if initResult.ProtocolVersion != protocolVersion {
		return &ProtocolError{
			Extension: "",
			Method:    "initialize",
			Err:       fmt.Errorf("protocol version mismatch: expected %q, got %q", protocolVersion, initResult.ProtocolVersion),
		}
	}

	// Send initialized notification (no response expected).
	if err := c.sendNotification(ctx, "notifications/initialized"); err != nil {
		return fmt.Errorf("dynamic initialized notification: %w", err)
	}

	return nil
}

// ListTools calls extensions/list to discover available tools.
func (c *Client) ListTools(ctx context.Context) ([]ToolDef, error) {
	var allTools []ToolDef
	cursor := ""

	for {
		params := map[string]interface{}{}
		if cursor != "" {
			params["cursor"] = cursor
		}

		var raw json.RawMessage
		if err := c.sendRequest(ctx, "extensions/list", params, &raw); err != nil {
			return nil, fmt.Errorf("dynamic extensions/list: %w", err)
		}

		var result listToolsResult
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, fmt.Errorf("dynamic extensions/list: parse result: %w", err)
		}

		allTools = append(allTools, result.Tools...)

		if result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}

	return allTools, nil
}

// CallTool invokes a tool by name with the given JSON arguments.
func (c *Client) CallTool(ctx context.Context, name string, args json.RawMessage) (string, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": json.RawMessage(args),
	}

	var raw json.RawMessage
	if err := c.sendRequest(ctx, "extensions/call", params, &raw); err != nil {
		return "", fmt.Errorf("dynamic extensions/call %q: %w", name, err)
	}

	var result callToolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("dynamic extensions/call %q: parse result: %w", name, err)
	}

	if result.IsError {
		texts := make([]string, 0, len(result.Content))
		for _, c := range result.Content {
			if c.Type == "text" {
				texts = append(texts, c.Text)
			}
		}
		return "", fmt.Errorf("extension tool error: %v", texts)
	}

	texts := make([]string, 0, len(result.Content))
	for _, c := range result.Content {
		if c.Type == "text" {
			texts = append(texts, c.Text)
		}
	}
	return joinTexts(texts), nil
}

// Shutdown sends a shutdown notification and waits for the process to exit.
// If the process does not exit within the given timeout, it is killed.
func (c *Client) Shutdown(ctx context.Context) error {
	// Send shutdown notification (best effort).
	_ = c.sendNotification(ctx, "extensions/shutdown")

	// Wait for process with timeout.
	done := make(chan error, 1)
	go func() {
		if c.cmd != nil && c.cmd.Process != nil {
			done <- c.cmd.Wait()
		} else {
			done <- nil
		}
	}()

	select {
	case <-done:
		// Process exited gracefully.
	case <-ctx.Done():
		// Timeout — kill the process.
		c.mu.Lock()
		if c.cmd != nil && c.cmd.Process != nil {
			c.cmd.Process.Kill()
		}
		c.mu.Unlock()
		<-done
	}

	return nil
}

// Close shuts down the subprocess abruptly (for cleanup).
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.closed = true
		if c.stdin != nil {
			c.stdin.Close()
		}
		if c.stdout != nil {
			c.stdout.Close()
		}
		if c.cmd != nil && c.cmd.Process != nil {
			c.cmd.Process.Kill()
			c.cmd.Wait()
		}
	})
	return nil
}

// sendRequest sends a JSON-RPC 2.0 request and waits for the matching response.
func (c *Client) sendRequest(ctx context.Context, method string, params interface{}, result interface{}) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client is closed")
	}
	c.nextID++
	id := c.nextID
	c.mu.Unlock()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	// Watch context and close stdout to unblock scanner on cancellation.
	ctxDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			c.mu.Lock()
			if c.stdout != nil {
				c.stdout.Close()
			}
			c.mu.Unlock()
		case <-ctxDone:
		}
	}()

	// Wait for the matching response.
	var readErr error
	for {
		if err := ctx.Err(); err != nil {
			close(ctxDone)
			return err
		}

		if !c.scanner.Scan() {
			if serr := c.scanner.Err(); serr != nil {
				readErr = fmt.Errorf("read response: %w", serr)
			} else {
				readErr = fmt.Errorf("server closed connection")
			}
			close(ctxDone)
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return readErr
		}

		line := c.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			// Malformed response — skip and continue scanning.
			continue
		}

		// Skip notifications (no ID) or responses to other requests.
		if resp.ID != id {
			continue
		}

		close(ctxDone)

		if resp.Error != nil {
			return fmt.Errorf("jsonrpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		if result != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("unmarshal result: %w", err)
			}
		}

		return nil
	}
}

// sendNotification sends a JSON-RPC 2.0 notification (no response expected).
func (c *Client) sendNotification(ctx context.Context, method string) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client is closed")
	}
	c.mu.Unlock()

	notif := struct {
		JSONRPC string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params,omitempty"`
	}{
		JSONRPC: "2.0",
		Method:  method,
	}

	data, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}
	data = append(data, '\n')

	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("write notification: %w", err)
	}
	return nil
}

// joinTexts concatenates text content parts with a newline separator.
func joinTexts(texts []string) string {
	if len(texts) == 0 {
		return ""
	}
	if len(texts) == 1 {
		return texts[0]
	}
	result := texts[0]
	for _, t := range texts[1:] {
		result += "\n" + t
	}
	return result
}
