# mcp-manager Specification

## Purpose

Manages the lifecycle of all configured MCP servers: starting, connecting, discovering tools, and shutting down. Provides a unified interface for the agent loop to access MCP tools.

## Requirements

### Requirement: REQ-MCP-11 — MCPManager Lifecycle

The `MCPManager` MUST manage the lifecycle of all configured MCP servers. It MUST maintain a registry of connected servers and their discovered tools.

#### Scenario: Manager created with config

- GIVEN a loaded MCP config with 2 servers
- WHEN MCPManager is created
- THEN the manager knows about the 2 servers but they are not yet connected

#### Scenario: Manager tracks connected servers

- GIVEN a manager with servers connected
- WHEN querying connected servers
- THEN only successfully connected servers are listed

### Requirement: REQ-MCP-12 — ConnectAll

The manager MUST provide a `ConnectAll()` method that starts and connects all enabled servers concurrently. It MUST initialize each server (JSON-RPC handshake) and discover tools via `tools/list`.

#### Scenario: All servers connect successfully

- GIVEN 3 enabled servers in config
- WHEN ConnectAll is called
- THEN all 3 servers are connected and their tools discovered

#### Scenario: Some servers fail to connect

- GIVEN 2 enabled servers where one fails to start
- WHEN ConnectAll is called
- THEN the successful server is connected
- AND the failed server is logged with error
- AND ConnectAll does not return an error (non-fatal)

### Requirement: REQ-MCP-13 — Shutdown

The manager MUST provide a `Shutdown()` method that stops all connected servers. For each server: kill the subprocess, close the connection, and clean up resources. Shutdown MUST be idempotent.

#### Scenario: Clean shutdown

- GIVEN a manager with connected servers
- WHEN Shutdown is called
- THEN all subprocesses are killed
- AND all connections are closed

#### Scenario: Shutdown with crashed server

- GIVEN a manager where one server already crashed
- WHEN Shutdown is called
- THEN the remaining servers are shut down
- AND no error is returned for the crashed server

### Requirement: REQ-MCP-14 — Tool Access

The manager MUST provide a `Tools()` method that returns all discovered MCP tools as `core.Tool` implementations. Tools from different servers MUST be distinguishable.

#### Scenario: Tools from multiple servers

- GIVEN servers "github" and "slack" each with 2 tools
- WHEN Tools() is called
- THEN 4 tools are returned
- AND each tool name is prefixed with its server name

#### Scenario: No servers connected

- GIVEN a manager with no connected servers
- WHEN Tools() is called
- THEN an empty list is returned

### Requirement: REQ-MCP-15 — Non-Fatal Server Failure

Server failure MUST be non-fatal. If one server crashes or fails to connect, other servers MUST continue operating normally. The failed server's tools MUST be marked unavailable.

#### Scenario: One server crashes mid-session

- GIVEN 3 connected servers
- WHEN server 2 crashes
- THEN servers 1 and 3 remain operational
- AND tools from server 2 return error on execution

#### Scenario: Server fails to connect

- GIVEN 2 enabled servers where one fails during ConnectAll
- WHEN ConnectAll completes
- THEN the successful server is connected
- AND the failed server's tools are not available
- AND subsequent tool calls to the failed server return error

### Requirement: REQ-MCP-16 — Tool Name Prefixing

MCP tool names MUST be prefixed with the server name and underscore: `{serverName}_{toolName}`. This prevents collisions between built-in and MCP tools, and between MCP tools from different servers.

#### Scenario: Prefixed tool names

- GIVEN server "github" with tool "create_issue"
- WHEN the tool is registered
- THEN the full name is "github_create_issue"

#### Scenario: Collision avoidance

- GIVEN built-in tool "read_file" and MCP server "filetools" with tool "read_file"
- WHEN both are registered
- THEN MCP tool is "filetools_read_file" and built-in remains "read_file"
