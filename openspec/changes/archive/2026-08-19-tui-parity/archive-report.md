# Archive Report: TUI Parity

Change: `tui-parity`
Archived: 2026-08-19
Status: COMPLETE (code + tests green; uncommitted working tree)

## Summary
Removed all fabricated/hardcoded UI data from kui's TUI and aligned rendering to
OpenCode's *actual* (dynamic) behavior. Every visible field is now either sourced
from a real controller method (`TotalTokens`, `Cost`, `ModelName`, `ActiveProfile`)
or omitted when kui has no data source (MCP/LSP status, app version).

## What changed
| File | Change |
|------|--------|
| `internal/tui/views/footer.go` | Removed `OpenCode 1.18.18` + MCP/LSP dots; render real tokens/cost/model + `ctrl+p commands` |
| `internal/tui/views/sidebar.go` | Removed `version`, `Subagents` fake, `MCP`/`LSP` fakes; Context real; theme tokens for header/bg |
| `internal/tui/views/model_list.go` | Removed `mimo-*` fakes + dead `modelProvider`; empty fallback when no provider |
| `internal/tui/views/chat.go` | `#808080` → `styles.HomeMuted` |
| `internal/tui/views/command_palette.go` | Inline border/bg → `styles.Popup`; threads `a.styles` |
| `internal/tui/app.go` | Pass `a.styles` to command palette |
| `internal/tui/views/footer_test.go` | Dropped fake-pinning MCP assertions |
| `internal/tui/app_test.go` | `ctrl+p commands` instead of `MCP` |
| `internal/tui/views/footer_lsp_test.go` | Deleted (pinned removed fakes) |
| `internal/tui/views/parity_test.go` | NEW: RED tests asserting fakes absent |
| `openspec/changes/tui-parity/*` | SDD artifacts (proposal/spec/design/tasks/verify/sync) |

## Verification
- `go test ./...` → pass
- `go vet ./internal/tui/...` → clean
- Forbidden literals → zero in `views/*.go`

## Lessons
- The prior `tui-redesign` shipped invented strings (version, token counts, provider
  names, status dots) with no backend and no OpenCode equivalent. This change treats
  "rendered only if a real source exists" as a hard requirement.
- Subagents were unavailable (API 503); exploration and implementation were done
  inline by the orchestrator with strict TDD.

## Open actions (not blocking)
1. Maintainers must `gentle-ai sdd-attempt reset` for `tui-parity` to close the
   native work-unit ledger (settle returned `maintainer_decision`).
2. Commit requires explicit authorization + the lifecycle review gate.
