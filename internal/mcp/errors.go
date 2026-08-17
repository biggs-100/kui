package mcp

import "fmt"

// MCPConnectionError wraps failures connecting to or communicating with an
// MCP server subprocess. It names the server for clear diagnostics.
type MCPConnectionError struct {
	Server string
	Err    error
}

func (e *MCPConnectionError) Error() string {
	return fmt.Sprintf("mcp server %q connection failed: %v", e.Server, e.Err)
}

// Unwrap exposes the underlying cause to errors.Is / errors.As.
func (e *MCPConnectionError) Unwrap() error {
	return e.Err
}

// MCPToolError wraps failures from an MCP tool execution (tools/call). It
// carries both the server name and tool name for traceability.
type MCPToolError struct {
	Server string
	Tool   string
	Err    error
}

func (e *MCPToolError) Error() string {
	return fmt.Sprintf("mcp tool %q on server %q failed: %v", e.Tool, e.Server, e.Err)
}

// Unwrap exposes the underlying cause to errors.Is / errors.As.
func (e *MCPToolError) Unwrap() error {
	return e.Err
}

// MCPVersionError is returned when the server's protocol version does not
// match the client's expected version during initialize handshake.
type MCPVersionError struct {
	Server   string
	Expected string
	Got      string
}

func (e *MCPVersionError) Error() string {
	return fmt.Sprintf("mcp server %q protocol version mismatch: expected %q, got %q", e.Server, e.Expected, e.Got)
}

// MCPConfigError is returned when an MCP configuration is invalid.
type MCPConfigError struct {
	File  string
	Field string
	Err   error
}

func (e *MCPConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("mcp config %q: field %q: %v", e.File, e.Field, e.Err)
	}
	return fmt.Sprintf("mcp config %q: %v", e.File, e.Err)
}

// Unwrap exposes the underlying cause to errors.Is / errors.As.
func (e *MCPConfigError) Unwrap() error {
	return e.Err
}
