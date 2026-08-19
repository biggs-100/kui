# Proposal: LSP Integration

## Intent

Kui agents write Go code but lack real-time language intelligence. Errors, type info, and symbol navigation require manual `go vet`/`go build` round-trips. LSP integration gives agents program-level awareness — diagnostics inline, hover for types, go-to-definition and references for navigation — directly in the TUI.

## Scope

### In Scope
- LSP client package (`internal/lsp/`) following MCP client pattern (JSON-RPC 2.0, stdio, handshake)
- `textDocument/publishDiagnostics` — push-based error/warning cache, inline display in TUI
- `textDocument/hover` — type info and doc comments for agent context
- `textDocument/definition` — navigate to symbol definition
- `textDocument/references` — find all usages of a symbol
- File sync: `didOpen`/`didChange`/`didClose` notifications on `write_file` and `read_file`
- Graceful degradation when gopls is not installed (warn, don't fail)
- Lazy startup (background, non-blocking)
- Go-only MVP (gopls)

### Out of Scope
- Completion (`textDocument/completion`) — high complexity, TUI is chat-driven
- Code actions / quick fixes — medium value, high complexity
- Rename refactoring — low value for agent TUI
- Multi-language support — deferred to Phase 2
- LSP server configuration UI — use gopls defaults

## Capabilities

### New Capabilities
- `lsp-client`: LSP client with JSON-RPC 2.0 transport, initialize handshake, file sync, and server lifecycle
- `lsp-diagnostics`: Push-based diagnostic cache with severity filtering and TUI inline display
- `lsp-tools`: Four agent tools — `lsp_hover`, `lsp_definition`, `lsp_references`, `lsp_diagnostics`

### Modified Capabilities
- `agent-tools`: LSP tools MUST be registered alongside built-in and MCP tools in the tool registration surface
- `tui-chat`: Chat view MUST render inline diagnostic annotations (errors/warnings) below affected lines

## Approach

Mirror the MCP client architecture exactly: separate `LSPClient` (transport + protocol), `LSPManager` (lifecycle + caching), and tool wrappers implementing `core.Tool`. One gopls instance per workspace. Startup is lazy — first LSP tool call triggers initialization. Diagnostics are cached server-side and updated on `publishDiagnostics`. File sync notifications fire from `write_file` and `read_file` tool hooks.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/lsp/` | New | LSP client, manager, diagnostics cache, tools |
| `internal/core/tool.go` | Modified | Register LSP tools in tool surface |
| `internal/tui/chat.go` | Modified | Render inline diagnostic annotations |
| `internal/runtime/` | Modified | Wire LSPManager into composition root |
| `go.mod` | Modified | Add go-lsp/lspclient dependency |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| gopls not installed | High | Detect via `exec.LookPath`, warn gracefully, LSP tools return "gopls not found" |
| Startup latency (1-3s) | Medium | Lazy init in background goroutine, tools block only on first call |
| File sync drift | Medium | Hook into write_file/read_file to fire didChange notifications |
| gopls crash | Low | Detect subprocess exit, mark tools unavailable, auto-restart on next call |

## Rollback Plan

Remove `internal/lsp/` package, delete LSP tool registrations from `internal/core/tool.go`, remove diagnostic rendering from `internal/tui/chat.go`, revert `go.mod`/`go.sum`. No data migration needed — LSP state is ephemeral per session.

## Dependencies

- `github.com/go-lsp/lspclient` v1.x (MIT, Go 1.18+, pure Go)
- `github.com/go-lsp/protocol` v1.x (MIT, types only)
- `gopls` installed on user's PATH (recommended, not required)

## Success Criteria

- [ ] Agent can call `lsp_hover` and receive type info for a Go symbol
- [ ] Agent can call `lsp_definition` and navigate to a symbol's definition file/line
- [ ] Agent can call `lsp_references` and list all usages of a symbol
- [ ] Agent can call `lsp_diagnostics` and see file errors/warnings
- [ ] Diagnostics render inline in the TUI below affected lines
- [ ] `write_file` tool triggers `didChange` notification to gopls
- [ ] When gopls is not installed, all LSP tools return a clear error (no crash)
- [ ] Startup is non-blocking — TUI remains responsive during gopls init
- [ ] `go test ./...` passes with all new and existing tests
