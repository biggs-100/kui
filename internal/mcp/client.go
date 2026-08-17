package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

const protocolVersion = "2025-03-26"

// MCPToolDef describes a tool returned by an MCP server's tools/list response.
type MCPToolDef struct {
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

// listToolsResult is the shape of the tools/list result.
type listToolsResult struct {
	Tools      []MCPToolDef `json:"tools"`
	NextCursor string       `json:"nextCursor,omitempty"`
}

// callToolResult is the shape of the tools/call result.
type callToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError,omitempty"`
}

// Client communicates with an MCP server subprocess via JSON-RPC 2.0 over stdio.
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

// NewClient spawns the MCP server subprocess and prepares the stdio transport.
// The command array is passed directly to os/exec. cwd and env may be empty.
func NewClient(ctx context.Context, command []string, cwd string, env map[string]string) (*Client, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("mcp client: command is required")
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = cwd

	if env != nil {
		cmd.Env = make([]string, 0, len(env))
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp client: stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp client: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp client: start: %w", err)
	}

	return &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		scanner: bufio.NewScanner(stdout),
	}, nil
}

// Initialize sends the MCP initialize handshake (REQ-MCP-6). It must be
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
		return fmt.Errorf("mcp initialize: %w", err)
	}

	// Send initialized notification (no response expected)
	if err := c.sendNotification(ctx, "notifications/initialized"); err != nil {
		return fmt.Errorf("mcp initialized notification: %w", err)
	}

	return nil
}

// ListTools calls tools/list to discover available tools (REQ-MCP-7).
func (c *Client) ListTools(ctx context.Context) ([]MCPToolDef, error) {
	var allTools []MCPToolDef
	cursor := ""

	for {
		params := map[string]interface{}{}
		if cursor != "" {
			params["cursor"] = cursor
		}

		var raw json.RawMessage
		if err := c.sendRequest(ctx, "tools/list", params, &raw); err != nil {
			return nil, fmt.Errorf("mcp tools/list: %w", err)
		}

		var result listToolsResult
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, fmt.Errorf("mcp tools/list: parse result: %w", err)
		}

		allTools = append(allTools, result.Tools...)

		if result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}

	return allTools, nil
}

// CallTool invokes a tool by name with the given JSON arguments (REQ-MCP-8).
func (c *Client) CallTool(ctx context.Context, name string, args json.RawMessage) (string, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": json.RawMessage(args),
	}

	var raw json.RawMessage
	if err := c.sendRequest(ctx, "tools/call", params, &raw); err != nil {
		return "", fmt.Errorf("mcp tools/call %q: %w", name, err)
	}

	var result callToolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("mcp tools/call %q: parse result: %w", name, err)
	}

	if result.IsError {
		texts := make([]string, 0, len(result.Content))
		for _, c := range result.Content {
			if c.Type == "text" {
				texts = append(texts, c.Text)
			}
		}
		return "", &MCPToolError{Tool: name, Err: fmt.Errorf("tool error: %v", texts)}
	}

	// Concatenate all text content (REQ-MCP-8).
	texts := make([]string, 0, len(result.Content))
	for _, c := range result.Content {
		if c.Type == "text" {
			texts = append(texts, c.Text)
		}
	}
	return joinTexts(texts), nil
}

// Close shuts down the subprocess gracefully.
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
			// If context is cancelled, return context error.
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
			// Malformed response — skip and continue scanning (REQ-MCP-5).
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
