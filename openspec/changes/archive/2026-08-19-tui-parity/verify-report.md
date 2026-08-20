# Verify Report: TUI Parity

Change: `tui-parity`
Status: VERIFIED (green)

## Evidence

- `go test ./...` → all packages OK (incl. `internal/tui`, `internal/tui/views`).
- `go vet ./internal/tui/...` → clean.
- `gofmt -l` → files touched are formatted (repo had pre-existing unformatted files;
  only modified files were formatted).
- Forbidden-literal grep in `internal/tui/views/*.go` (excluding comments and the
  parity_test assertions) → zero matches for `1.18.18`, `1.2.1`, `context7`,
  `engram`, `319k`, `mimo`.

## Strict TDD (RED → GREEN)

1. **RED** — added `internal/tui/views/parity_test.go` asserting footer/sidebar/model
   catalog do NOT contain the fabricated strings. Ran `go test -run TestParity` → FAIL
   (fakes present), confirming the baseline bug.
2. **GREEN** — removed fakes:
   - `footer.go`: dropped version block (`OpenCode 1.18.18 ●`) and MCP/LSP dots; footer
     now shows only `dir • tokens (pct%) · $cost • model • ctrl+p commands` from real
     `controller.TotalTokens()`/`Cost()`/`ModelName()`.
   - `sidebar.go`: removed `version` field, `Subagents` fake block, `MCP`/`LSP` fake
     sections; `Context` shows real tokens/cost or truthful `0 tokens 0% $0.00`;
     header uses `styles.LogoAccent`, background uses `styles.Sidebar`.
   - `model_list.go`: removed `mimo-v2-free`/`mimo-v2.5`/`mimo-v2.5-free` from
     `AvailableModels()` and the `modelProvider` map; `AvailableModelsFiltered()` returns
     empty (not the static catalog) when no provider is configured.
   - `chat.go`, `command_palette.go`: replaced inline hex literals with theme tokens
     (`HomeMuted`, `Popup`); `command_palette` now threads `a.styles` from `app.go`.
3. **Refactor** — removed dead code: `modelProvider` map, `FooterModel.SetMCPStatus`/
   `SetLSPStatus`, `footer_lsp_test.go` (pinned the removed fakes).
4. Re-ran `go test -run TestParity` → PASS.

## Tests updated (were pinning fakes)
- `footer_test.go`: `TestFooterRenderFull` no longer asserts `MCP connected`; removed
  `TestFooterMCPStatus` (tested removed field).
- `app_test.go`: `TestAppViewIncludesFooter` asserts `ctrl+p commands` instead of `MCP`.
- Deleted `footer_lsp_test.go` (exercised `SetMCPStatus`/`SetLSPStatus`, now gone).

## Spec conformance
- REQ-TUI-PARITY-1 ✓ no version strings
- REQ-TUI-PARITY-2 ✓ tokens/cost from controller
- REQ-TUI-PARITY-3 ✓ no MCP/LSP dots without data
- REQ-TUI-PARITY-4 ✓ no fabricated model IDs; empty fallback
- REQ-TUI-PARITY-5 ✓ no hex literals in `views/` (theme tokens used)
- REQ-TUI-PARITY-6 ✓ Subagents/version block removed

## Open items
- SDD attempt ledger `settle` returned `blocked`/`maintainer_decision`: the native
  work-unit budget needs a maintainer `reset` to close accounting. Code + tests are
  complete and green; this is harness bookkeeping only.
- Commit NOT performed (no explicit commit authorization this session; lifecycle
  review gate applies).
