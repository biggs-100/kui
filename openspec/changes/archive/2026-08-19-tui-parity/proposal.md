# Proposal: TUI Parity — Remove hardcoded fakes, align to OpenCode truth

## Intent

The previous `tui-redesign` shipped UI strings and data that **do not exist** in
OpenCode and are not backed by any runtime source in kui. This change removes the
fabricated data and makes every rendered field either dynamic (from a real
controller source) or absent (when kui has no such data). The goal is parity with
OpenCode's *actual* behavior, not invention.

## Why (evidence)

Mapped OpenCode real TUI (`C:/Users/USER/Desktop/herramientas/opencode/packages/tui/src`)
vs kui source. Confirmed fakes:

| Fake in kui | OpenCode truth | kui source of fake |
|---|---|---|
| `OpenCode 1.18.18` (footer) | Dynamic `InstallationVersion`, no literal | `internal/tui/views/footer.go:149` |
| `v1.2.1` (sidebar) | No version badge; dynamic version | `internal/tui/views/sidebar.go:29,109` |
| `319k tokens 32% $0.27` | Real `{tokens} tokens` + `{pct}%` + `{money}` | `internal/tui/views/sidebar.go:130` fallback |
| `context7 • engram` + `○ disconnected` | MCP listed generically; per-workspace status | `internal/tui/views/sidebar.go:182` |
| `v1.2.1 • 0 run • 0 done • Σ 0` | No subagent run counter UI | `internal/tui/views/sidebar.go:107` |
| `mimo-v2-free` / `mimo-v2.5` | No such models; catalog is live SDK | `internal/tui/views/model_list.go:46-60` |
| Literal `#1a1a1a` / `#569cd6` / `#252525` / `#4ec9b0` | Central `Theme` struct, no scattered literals | `views/sidebar.go`, `views/footer.go` |

**Real data kui already has** (wired, must be used instead of fakes):
- `controller.TotalTokens()` / `Cost()` — accumulated via `TrackUsage` on every
  `streamDoneMsg` (`app.go:161`). Real after a stream.
- `controller.ActiveProfile()` / `Profiles()` / `ModelName()` / `ContextWindow()` — real.
- `AvailableModelsFiltered()` — already does live `/models` discovery with static
  fallback (`model_list.go`). The fallback is the only place fakes appear.

**No data source exists** (must be removed, not faked):
- LSP status: `SetLSPStatus` has no caller; kui tracks no LSP server state.
- MCP status: `SetMCPStatus` has no caller; kui tracks no MCP server state.
- App version: kui has no version string (unlike OpenCode's `InstallationVersion`).

## Scope

### In Scope
1. **Footer** (`views/footer.go`) — drop `OpenCode 1.18.18`, drop fake token string;
   render real `tokens`/`cost` only when `> 0`, else a single `—`. Keep `directory`.
   Remove MCP/LSP dots (no data source). Remove `ctrl+p commands` hardcoded hint
   unless it maps to a real binding.
2. **Sidebar** (`views/sidebar.go`) — remove `version` field and `Subagents` fake
   block; remove `context7 • engram` / `○ disconnected`; Context section shows real
   `tokens`/`cost`/`contextMax` or `0 tokens 0% $0.00`; MCP/LSP sections removed
   (no data) or shown as `not available`.
3. **Model catalog** (`views/model_list.go`) — remove fabricated `mimo-*` entries
   from `AvailableModels()`; live discovery stays; when nothing is configured, show
   an empty/help state instead of fake models.
4. **Theme literals** — replace scattered hex literals in `views/sidebar.go`,
   `views/footer.go`, `views/header.go` (if any) with `theme.Styles` / `theme.Theme`
   values. Confirm `theme/opencode.go` exposes the needed tokens.

### Out of Scope
- Layout changes (home centering, sidebar width/breakpoint) — already correct from
  `tui-redesign`; adjust only if a fake removal forces it.
- New features (LSP/MCP live status, version reporting) — future changes.
- Color palette redesign — keep OpenCode gray; only de-literalize.

## Target Users
- kui users who expect truthful, OpenCode-faithful UI.
- Reviewers verifying no fabricated data remains.

## Affected Areas

| Area | Change |
|------|--------|
| `internal/tui/views/footer.go` | Remove version + fake tokens + MCP/LSP dots |
| `internal/tui/views/sidebar.go` | Remove version, Subagents fake, MCP/LSP fakes, literals |
| `internal/tui/views/model_list.go` | Remove `mimo-*` fakes; empty-state fallback |
| `internal/tui/theme/opencode.go` | Ensure tokens needed by views exist |
| `internal/tui/app.go` | Stop calling `SetMCPStatus`/`SetLSPStatus` if they feed fakes (verify) |

## Current State Gap
kui renders invented version numbers, token counts, provider names, and status dots
that have no backend and do not match OpenCode. After this change every visible
field is either real or omitted.

## Success Criteria
1. `footer.go` / `sidebar.go` contain no `1.18.18`, `1.2.1`, `context7`, `engram`,
   `mimo`, `319k`, or hardcoded semver.
2. Tokens/cost render only from `controller.TotalTokens()` / `Cost()`.
3. `/model` shows live models only; no `mimo-*` entries ever appear.
4. No hex literals remain in `views/*.go` outside `theme` package.
5. `go test ./...` passes (strict TDD).
6. `gofmt` / `go vet` clean.

## Rollback Plan
Single commit; revert via `git revert` if regression. No schema/migration changes.
