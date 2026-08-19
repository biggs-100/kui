# lsp-client Specification

## Purpose

LSP client package (`internal/lsp/`) providing JSON-RPC 2.0 transport over stdio, LSP protocol handshake, server lifecycle management, and file synchronization. Follows the MCP client pattern (REQ-MCP-5 through REQ-MCP-10) for transport and lifecycle, adapted for the LSP protocol.

## Requirements

### Requirement: REQ-LSP-1 — JSON-RPC 2.0 Transport

The client MUST communicate with LSP servers using JSON-RPC 2.0 over the subprocess's stdin (requests) and stdout (responses). Messages MUST use Content-Length framing per the LSP base protocol, NOT newline-delimited as MCP does.

#### Scenario: Send request and receive response

- GIVEN a running gopls subprocess
- WHEN the client sends a JSON-RPC request on stdin with Content-Length header
- THEN a JSON-RPC response is received on stdout with matching Content-Length

#### Scenario: Server notification received

- GIVEN a running gopls subprocess
- WHEN the server sends a notification (e.g., `textDocument/publishDiagnostics`)
- THEN the client parses and dispatches it to registered handlers

### Requirement: REQ-LSP-2 — Initialize Handshake

The client MUST send an `initialize` request with `processId`, `rootUri`, and `capabilities` before any other requests. The client MUST send `initialized` notification after receiving the initialize response. The client MUST NOT send other requests until the handshake completes.

#### Scenario: Successful handshake

- GIVEN a gopls subprocess on PATH
- WHEN the client calls initialize with rootUri
- THEN the response contains server capabilities
- AND the client sends `initialized` notification
- AND subsequent requests are allowed

#### Scenario: gopls not found

- GIVEN gopls is not installed or not on PATH
- WHEN the client attempts to start the server
- THEN the client returns a "gopls not found" error
- AND no subprocess is started

### Requirement: REQ-LSP-3 — Server Lifecycle

The client MUST support start, stop, and restart operations. Start MUST launch the gopls subprocess. Stop MUST send shutdown request then exit notification, then kill the subprocess. Restart MUST stop then start. The client MUST track server state (stopped, starting, running, error).

#### Scenario: Start server

- GIVEN a stopped LSP client
- WHEN start is called with server path and args
- THEN gopls subprocess is launched
- AND state transitions to "running" after handshake

#### Scenario: Stop server

- GIVEN a running LSP client
- WHEN stop is called
- THEN shutdown and exit are sent
- AND the subprocess is terminated
- AND state transitions to "stopped"

#### Scenario: Restart server

- GIVEN a running LSP client with stale state
- WHEN restart is called
- THEN the server stops and starts fresh
- AND state transitions through "stopped" → "starting" → "running"

### Requirement: REQ-LSP-4 — Configuration

The client MUST accept configuration: server path (default: "gopls" from PATH), server arguments (default: `["-logfile=/dev/null"]`), and rootUri (workspace root). Configuration MUST be immutable after start — changing config requires restart.

#### Scenario: Default configuration

- GIVEN no explicit configuration
- WHEN the client starts
- THEN it uses "gopls" from PATH with default args
- AND rootUri defaults to the workspace root

#### Scenario: Custom configuration

- GIVEN server path "/usr/local/bin/gopls" and args `["-remote=auto"]`
- WHEN the client starts
- THEN it launches the specified binary with the given args

### Requirement: REQ-LSP-5 — Error Handling and Graceful Degradation

The client MUST handle transport errors, server crashes, and protocol errors without panicking. On crash, the client MUST mark tools as unavailable and return clear error messages. The client MUST support auto-restart on next call after crash.

#### Scenario: Server crash during request

- GIVEN a running gopls subprocess
- WHEN the server subprocess exits during a pending request
- THEN the client returns a "server exited" error
- AND state transitions to "error"

#### Scenario: Auto-restart after crash

- GIVEN a client in "error" state
- WHEN a new request arrives
- THEN the client attempts to restart the server
- AND if restart succeeds, the request proceeds

### Requirement: REQ-LSP-6 — File Sync Notifications

The client MUST send `textDocument/didOpen` when a file is first accessed, `textDocument/didChange` when content changes, and `textDocument/didClose` when a file is no longer tracked. File sync MUST track open file versions.

#### Scenario: Open file

- GIVEN a file not yet tracked
- WHEN didOpen is sent with file content
- THEN the server is notified of the open file
- AND the file is tracked with version 1

#### Scenario: Change file

- GIVEN an open tracked file at version 1
- WHEN didChange is sent with new content
- THEN the server receives the change
- AND the file version increments to 2

#### Scenario: Close file

- GIVEN an open tracked file
- WHEN didClose is sent
- THEN the server is notified
- AND the file is removed from the tracked set
