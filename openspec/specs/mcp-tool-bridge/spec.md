# mcp-tool-bridge Specification

## Purpose

Bridges MCP tools to the kui core tool interface. Wraps an MCP tool definition (name, description, schema) and delegates execution to the MCP client.

## Requirements

### Requirement: REQ-MCP-17 — MCPTool Interface

`MCPTool` MUST implement the `core.Tool` interface: `Name() string`, `Description() string`, `Schema() map[string]any`, `Execute(input map[string]any) (string, error)`. The tool's metadata MUST come from the `tools/list` response.

#### Scenario: Tool metadata from MCP

- GIVEN an MCP tool definition with name "search", description "Search docs", schema {query: string}
- WHEN MCPTool is created
- THEN Name() returns "search"
- AND Description() returns "Search docs"
- AND Schema() returns the parameter schema

#### Scenario: Prefixed name

- GIVEN MCP tool "search" from server "docs"
- WHEN MCPTool is created via MCPManager
- THEN Name() returns "docs_search"

### Requirement: REQ-MCP-18 — Execute via JSON-RPC

`MCPTool.Execute()` MUST send a `tools/call` JSON-RPC request to the server with the tool name and arguments. The request MUST include the original (unprefixed) tool name.

#### Scenario: Successful execution

- GIVEN an MCPTool for "docs_search"
- WHEN Execute is called with { query: "test" }
- THEN a tools/call request is sent with name "search" and args { query: "test" }

#### Scenario: Execution with no arguments

- GIVEN an MCPTool that takes no parameters
- WHEN Execute is called with empty input
- THEN a tools/call request is sent with empty args

### Requirement: REQ-MCP-19 — Response Handling

The `content[].text` field from the `tools/call` response MUST be returned as the string result. Multiple content items MUST be concatenated.

#### Scenario: Single text content

- GIVEN a tools/call response with content: [{ type: "text", text: "result" }]
- WHEN the response is processed
- THEN Execute returns "result"

#### Scenario: Multiple text contents

- GIVEN a tools/call response with content: [{ type: "text", text: "line1" }, { type: "text", text: "line2" }]
- WHEN the response is processed
- THEN Execute returns "line1\nline2"

### Requirement: REQ-MCP-20 — Error Response

When the `tools/call` response has `isError: true`, the client MUST return an error to the caller. The error message SHOULD include the content text.

#### Scenario: Server-side error

- GIVEN a tools/call response with isError: true and content: [{ type: "text", text: "tool failed" }]
- WHEN the response is processed
- THEN Execute returns an error containing "tool failed"

#### Scenario: Network error

- GIVEN a network failure during tools/call
- WHEN the call is made
- THEN Execute returns an error indicating the connection failure
