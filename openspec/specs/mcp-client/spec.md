# mcp-client Specification

## Purpose

JSON-RPC 2.0 client that communicates with MCP servers over stdio (stdin/stdout of a subprocess). Handles protocol handshake, tool discovery, and tool execution.

## Requirements

### Requirement: REQ-MCP-5 — JSON-RPC 2.0 Transport

The client MUST communicate with MCP servers using JSON-RPC 2.0 over the subprocess's stdin (requests) and stdout (responses). Each message MUST be a single line (no Content-Length framing).

#### Scenario: Send request and receive response

- GIVEN a running MCP server subprocess
- WHEN the client sends a JSON-RPC request on stdin
- THEN a JSON-RPC response is received on stdout

#### Scenario: Malformed response handling

- GIVEN a server that returns non-JSON on stdout
- WHEN the client reads the response
- THEN the client returns a parse error (not panic)

### Requirement: REQ-MCP-6 — Initialize Handshake

The client MUST send an `initialize` request with `protocolVersion: "2025-03-26"` before any other requests. The server MUST respond with its capabilities. The client MUST NOT proceed until initialize completes.

#### Scenario: Successful handshake

- GIVEN a server supporting protocol version "2025-03-26"
- WHEN the client calls initialize
- THEN the response contains server capabilities
- AND subsequent requests are allowed

#### Scenario: Version mismatch

- GIVEN a server supporting a different protocol version
- WHEN the client calls initialize
- THEN the client returns a version-mismatch error

### Requirement: REQ-MCP-7 — Tool Discovery

The client MUST call `tools/list` to discover available tools. The endpoint MAY be paginated with `nextCursor`. The client MUST fetch all pages until no nextCursor is returned.

#### Scenario: Single page of tools

- GIVEN a server with 3 tools
- WHEN tools/list is called
- THEN all 3 tools are returned (no cursor in response)

#### Scenario: Paginated tools

- GIVEN a server with tools returned in two pages
- WHEN tools/list is called with cursor
- THEN the client fetches both pages and returns the combined tool list

### Requirement: REQ-MCP-8 — Tool Execution

The client MUST call `tools/call` with the tool name and arguments. The response MUST include `content[]` with text results. The client MUST return the text content as the tool result.

#### Scenario: Successful tool call

- GIVEN a server with tool "echo"
- WHEN tools/call is invoked with `{ text: "hello" }`
- THEN the response content includes the echoed text

#### Scenario: Tool not found

- GIVEN a server without tool "nonexistent"
- WHEN tools/call is invoked for "nonexistent"
- THEN the response contains an error indicating tool not found

### Requirement: REQ-MCP-9 — Server Crash Handling

The client MUST handle server crashes gracefully: detect subprocess exit, return a clear error, and mark the server's tools as unavailable. The client MUST NOT panic.

#### Scenario: Server exits during tool call

- GIVEN a connected server
- WHEN the server subprocess exits during a tools/call
- THEN the client returns a "server exited" error
- AND the server's tools are marked unavailable

#### Scenario: Server exits before tool call

- GIVEN a server that crashed before a call
- WHEN tools/call is invoked
- THEN the client returns a "server not running" error

### Requirement: REQ-MCP-10 — Context Cancellation

The client MUST respect context cancellation. When a context is cancelled, the client MUST stop waiting for responses and clean up the subprocess.

#### Scenario: Cancel during handshake

- GIVEN a server that is slow to respond
- WHEN the context is cancelled during initialize
- THEN the client kills the subprocess and returns context error

#### Scenario: Cancel during tool call

- GIVEN a running tool call
- WHEN the context is cancelled
- THEN the client returns context error and cleans up
