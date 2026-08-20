# Tasks: TUI Parity

Change: `tui-parity`
Strict TDD: `go test ./...` (RED → GREEN → TRIANGULATE → REFACTOR)

Review workload forecast: ~6 files, ~200 changed lines. Chained PRs recommended: No.
400-line budget risk: Low. Decision needed before apply: No.

## Implementation tasks

- [x] T1 — Footer: remove fabricated version + status (REQ-1, REQ-3)
  - Delete `versionStr`/`versionDot` block in `views/footer.go` (lines ~146-149).
  - Delete `mcpStr`/`lspStr` computation and their `parts` entries.
  - Keep `dirStr`, `centerStr` (real tokens/cost), `modelStr`, `ctrl+p commands`.
  - `tokensStr`: when `m.tokens == 0` render `— tokens` (already `dash + " tokens"`);
    ensure cost shows `—` only when truly zero is acceptable — keep `$0.00` from real `Cost()`.
  - Replace removed `#4ec9b0` literal (gone with version block).

- [x] T2 — Sidebar: remove version + subagents + MCP/LSP fakes (REQ-1, REQ-3, REQ-6)
  - Remove `version` field; `NewSidebarModel` no longer sets `version`.
  - Remove `Subagents` section + `subLines` fake (`v%s • 0 run • 0 done • Σ 0`).
  - `Context` section: render real `m.tokens`/`m.cost`/`m.contextMax`; when 0 →
    `0 tokens 0% $0.00` (truthful zero). Remove `319k tokens 32% $0.27` branch.
  - Remove `MCP` section (`context7 • engram` / `○ disconnected`).
  - Remove `LSP` section (`○ disabled`).
  - Replace `headerStyle` literal `#569cd6` with `m.styles.LogoAccent` (bold).
  - Replace final `sidebarStyle.Background("#1a1a1a")` with `m.styles.Sidebar`.

- [x] T3 — Model catalog: drop fabricated IDs + empty fallback (REQ-4)
  - `AvailableModels()`: remove `mimo-v2-free`, `mimo-v2.5`.
  - `AvailableModelsFiltered()`: when `!anyConfigured` return `nil` (empty) so no
    fabricated list renders; caller shows help state.

- [x] T4 — De-literalize remaining view hex (REQ-5)
  - `views/chat.go:164,171`: `#808080` → `m.styles.HomeMuted`.
  - `views/command_palette.go:193-194`: use `m.styles.Popup` instead of inline
    `#333333`/`#1a1a1a`.

- [x] T5 — app.go: confirm no fake feeders
  - Verify `SetMCPStatus`/`SetLSPStatus` have no callers (grep already shows none).
  - No change needed unless a caller feeds fakes; document finding.

## Test tasks (strict TDD)

- [x] T6 — RED: add/extend tests asserting fakes are absent
  - `views/footer_test.go`: assert `Render()` does NOT contain `1.18.18`, `MCP`,
    `LSP`, `context7`, `engram`.
  - `views/sidebar_test.go`: assert `View()` does NOT contain `1.2.1`, `0 run`,
    `context7`, `engram`, `319k`, `disconnected`. Assert it DOES contain
    `0 tokens 0% $0.00` when no data.
  - `views/model_list_test.go`: assert `AvailableModels()` has no `mimo*`;
    `AvailableModelsFiltered()` returns empty when no provider configured.

- [x] T7 — Update existing tests that pin fakes
  - Existing `footer_test.go`/`sidebar_test.go` assertions on fake strings MUST be
    changed to assert absence (they currently encode the bug). This is required.

- [x] T8 — GREEN + verify
  - Run `go test ./...`; all pass.
  - `gofmt -l .` clean; `go vet ./...` clean.
  - Grep forbidden literals in `internal/tui/views/` → zero matches.

## Completion criteria
All checkboxes done; `go test ./...` green; no forbidden literal in `views/`.
