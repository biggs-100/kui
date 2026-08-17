package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// MCPTool implements core.Tool by delegating to an MCP client.
// Tool names are prefixed with the server name: {serverName}_{toolName}.
type MCPTool struct {
	name        string
	description string
	inputSchema string
	client      *Client
	serverName  string
	toolName    string
}

// NewMCPTool creates a new MCPTool from a tool definition and client reference.
// The name is prefixed with the server name per REQ-MCP-16.
func NewMCPTool(serverName string, def MCPToolDef, client *Client) *MCPTool {
	name := serverName + "_" + def.Name
	return &MCPTool{
		name:        name,
		description: def.Description,
		inputSchema: string(def.InputSchema),
		client:      client,
		serverName:  serverName,
		toolName:    def.Name,
	}
}

// Name returns the prefixed tool name: {serverName}_{toolName}.
func (t *MCPTool) Name() string {
	return t.name
}

// Description returns the tool's description from the MCP server.
func (t *MCPTool) Description() string {
	return t.description
}

// Schema returns the JSON parameter schema as a string.
func (t *MCPTool) Schema() string {
	return t.inputSchema
}

// Execute sends a tools/call request to the MCP server with the unprefixed tool name.
// Multiple text content items are concatenated with newlines (REQ-MCP-19).
func (t *MCPTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	result, err := t.client.CallTool(ctx, t.toolName, args)
	if err != nil {
		return "", &MCPToolError{
			Server: t.serverName,
			Tool:   t.toolName,
			Err:    fmt.Errorf("execute failed: %w", err),
		}
	}
	return result, nil
}
