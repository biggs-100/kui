# Tasks: MCP Client Support

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 500–600 (3 PRs) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| A | Config + JSON-RPC client | PR 1 (~250 lines) | `go test ./internal/mcp/ -run "TestConfig\|TestClient"` | Real subprocess: `echo` server mock | `internal/mcp/config.go`, `client.go`, `errors.go` + tests |
| B | Manager + Tool bridge | PR 2 (~200 lines) | `go test ./internal/mcp/ -run "TestManager\|TestMCPTool"` | Mock client (no real subprocess needed) | `internal/mcp/manager.go`, `tool.go` + tests |
| C | CLI integration | PR 3 (~100 lines) | `go test ./... && go build ./cmd/kui` | Full kui run with mock MCP in PATH | `cmd/kui/main.go`, `internal/tui/run.go` changes |

## Phase 1: Foundation — Config + Client (Slice A, PR 1)

- [x] 1.1 Create `internal/mcp/errors.go` — MCPConnectionError, MCPToolError types matching core error patterns
- [x] 1.2 RED: Create `internal/mcp/config_test.go` — test global-only load, project-overrides-global merge, empty config, unknown type rejection (REQ-MCP-1..4)
- [x] 1.3 GREEN: Create `internal/mcp/config.go` — ServerConfig struct, Config struct, Load(globalPath, projectPath) with YAML parsing and merge semantics
- [x] 1.4 REFACTOR: Extract merge logic, add comments on override semantics
- [x] 1.5 RED: Create `internal/mcp/client_test.go` — test initialize handshake, tools/list pagination, tools/call success, malformed response, version mismatch, server crash, context cancellation (REQ-MCP-5..10)
- [x] 1.6 GREEN: Create `internal/mcp/client.go` — NewClient, Initialize, ListTools, CallTool, Close; JSON-RPC 2.0 over stdio with bufio.Scanner, sync.Mutex ID matching, configurable timeout
- [x] 1.7 REFACTOR: DRY request/response helpers, add protocolVersion constant
- [x] 1.8 Verify: `go test ./internal/mcp/ -count=1` passes all config and client tests

## Phase 2: Core Implementation — Manager + Tool Bridge (Slice B, PR 2)

- [x] 2.1 RED: Create `internal/mcp/tool_test.go` — test Name() returns prefixed name, Description(), Schema() returns JSON string, Execute() calls tools/call with unprefixed name and returns concatenated text content, isError returns error (REQ-MCP-17..20)
- [x] 2.2 GREEN: Create `internal/mcp/tool.go` — MCPTool struct implementing core.Tool: Name (prefixed), Description, Schema (JSON string), Execute (delegates to client)
- [x] 2.3 REFACTOR: Extract response content concatenation helper
- [x] 2.4 RED: Create `internal/mcp/manager_test.go` — test NewManager with config, ConnectAll concurrent success + partial failure (non-fatal), Shutdown kills all + idempotent, Tools() returns prefixed tools, crashed server tools return error (REQ-MCP-11..16)
- [x] 2.5 GREEN: Create `internal/mcp/manager.go` — Manager struct, NewManager, ConnectAll (concurrent with errgroup), Shutdown (idempotent), Tools() returning []core.Tool
- [x] 2.6 REFACTOR: Add godoc on lifecycle states, tighten error logging
- [x] 2.7 Verify: `go test ./internal/mcp/ -count=1` passes all manager and tool tests

## Phase 3: Integration — CLI + TUI Wiring (Slice C, PR 3)

- [ ] 3.1 RED: Add test in `cmd/kui/` or integration test — verify MCP manager initializes from config, MCP tools appear in registry, shutdown cleans up (REQ-TOOLS-4 MCP tools included)
- [ ] 3.2 GREEN: Modify `cmd/kui/main.go` — load MCP config via `mcp.Load()`, create MCPManager, call ConnectAll (non-fatal on error), register MCP tools in full registry, defer Shutdown
- [ ] 3.3 GREEN: Modify `internal/tui/run.go` — add MCP initialization in TUI startup, register MCP tools before agent creation, defer shutdown
- [ ] 3.4 Verify: `go test ./... && go build ./cmd/kui` — all tests pass, guard test `TestCoreImportsStdlibOnly` still passes
- [ ] 3.5 Cleanup: Remove any dead code, verify gofmt/go vet clean
