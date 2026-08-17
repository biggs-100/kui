# Proposal: MCP Client Support

## Intent

kui tools are compiled-in only — every tool lives in `internal/adapters/tools/`. There's no way to use external tool servers (databases, APIs, domain-specific logic) without writing Go code and rebuilding. MCP (Model Context Protocol) is the emerging standard for AI agents to discover and invoke external tools via separate server processes. Adding MCP support gives kui access to the growing MCP ecosystem (hundreds of servers: GitHub, PostgreSQL, Slack, filesystem, etc.) and achieves parity with opencode which already supports MCP.

## Scope

### In Scope
- `mcp.yaml` config schema (global `~/.config/kui/mcp.yaml` + project `.kui/mcp.yaml`)
- MCP client: JSON-RPC 2.0 over stdio, stdlib-only (~200 lines Go)
- MCP manager: server lifecycle (connect, shutdown, tool cache)
- `MCPTool` adapter: implements `core.Tool`, bridges MCP tools to kui registry
- Tool discovery: `tools/list` → register in `core.Registry`
- Tool execution: `tools/call` → forward args, return result
- Permission integration: MCP tools follow existing `mcp_*` glob pattern

### Out of Scope
- HTTP/SSE transport (deferred — stdio covers 90% of use cases)
- Remote MCP servers (requires auth, different trust model)
- OAuth or token-based authentication
- MCP server installation/management (user provides server commands)
- Streaming tool results (MCP spec TBD)

## Capabilities

### New Capabilities
- `mcp-client`: JSON-RPC 2.0 client over stdio, server lifecycle management, tool discovery/execution bridge

### Modified Capabilities
- `agent-tools`: Registry accepts MCP-contributed tools alongside built-in tools
- `profile-permissions`: MCP tools follow existing `mcp_*` glob permission pattern (no spec change needed — already works)

## Approach

Hexagonal: core defines `MCPManager` port (connect, disconnect, list tools, call tool). Adapters implement stdio transport and JSON-RPC 2.0 framing.

Config-driven: `mcp.yaml` declares servers with command, args, env. Manager spawns subprocesses on startup, caches discovered tools, shuts down on exit.

Tool bridge: `MCPTool` wraps an MCP tool definition — Name/Description/Schema from `tools/list`, Execute calls `tools/call` via the manager's JSON-RPC client.

Stdlib-only core: JSON-RPC 2.0 is ~200 lines using `encoding/json`, `os/exec`, `bufio`. No external dependencies.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/mcp/` | New | Client, manager, config, MCPTool adapter |
| `internal/mcp/client.go` | New | JSON-RPC 2.0 over stdio |
| `internal/mcp/manager.go` | New | Server lifecycle, tool cache |
| `internal/mcp/config.go` | New | mcp.yaml parsing |
| `internal/mcp/tool.go` | New | MCPTool implements core.Tool |
| `internal/adapters/tools/registry.go` | Modified | Accept MCP tools in Default() |
| `cmd/kui/main.go` | Modified | Initialize MCP manager, register tools |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| MCP server crashes mid-session | Med | Manager detects process exit, returns clear error, marks tools unavailable |
| Tool name collisions (built-in vs MCP) | Low | Prefix MCP tools with server name: `github_create_issue` |
| Stdio framing errors | Low | Strict JSON-RPC 2.0 parsing with timeout on reads |
| Config missing or malformed | Low | Graceful degradation — skip MCP, log warning, continue with built-in tools |

## Rollback Plan

Remove `internal/mcp/`, revert `registry.go` and `main.go` changes. kui reverts to built-in tools only. No data migration — MCP is stateless runtime config.

## Dependencies

- None (pure Go, stdlib only)

## Success Criteria

- [ ] `mcp.yaml` config loads and parses correctly
- [ ] MCP client connects to a test server via stdio
- [ ] `tools/list` discovers tools and registers them in `core.Registry`
- [ ] `tools/call` executes an MCP tool and returns results
- [ ] Manager shuts down servers cleanly on exit
- [ ] MCP tools respect existing permission glob patterns
- [ ] `TestCoreImportsStdlibOnly` guard test still passes
- [ ] All existing tests pass unmodified
