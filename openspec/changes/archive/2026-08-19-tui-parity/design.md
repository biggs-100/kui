# Design: TUI Parity

## Approach

Remove fabrication at the source. Each view field becomes either (a) driven by a
real controller method, or (b) omitted when kui has no data. No new fake defaults.

## Data sources (authoritative)

| Field | Source | Wired? |
|-------|--------|--------|
| tokens | `controller.TotalTokens()` | yes (`TrackUsage` on `streamDoneMsg`) |
| cost | `controller.Cost()` | yes |
| model | `controller.ModelName()` | yes |
| profile | `controller.ActiveProfile()` | yes |
| context window | `controller.ContextWindow()` | yes |
| MCP status | none | **no caller** → omit |
| LSP status | none | **no caller** → omit |
| app version | none | **does not exist** → omit |
| models | `liveModelsForProvider` per provider | yes (live /models) |

## Component changes

### `views/footer.go`
- Delete `versionStr` / `versionDot` block (REQ-1).
- Delete `mcpStr` / `lspStr` (REQ-3).
- `tokensStr`: keep `if m.tokens > 0` branch using real `m.tokens`/`m.contextMax`;
  else render `— tokens`. Drop the `%d tokens (%d%%)` raw-only note.
- `costStr`: render `$%.2f` from `m.cost` (real). Show `—` when cost == 0? Keep
  `$0.00` only if it is genuinely the accumulated cost (it is). Acceptable.
- `modelStr`: keep (real `controller.ModelName`).
- `ctrl+p commands`: KEEP only if a real `ctrl+p` binding opens the command palette;
  otherwise drop. Verify against `app.go` keybindings in apply.
- Colors: replace `#4ec9b0` literal with `m.styles.StatusOK` (green dot) or remove
  the dot entirely. Use `theme` tokens.

### `views/sidebar.go`
- Remove `version` field + `NewSidebarModel` init `version: "1.2.1"` (REQ-1).
- Remove `Subagents` section + `subLines` fake (REQ-6).
- `Context` section: render real `m.tokens`/`m.cost`/`m.contextMax`; when 0 show
  `0 tokens 0% $0.00` (truthful zero, not `319k`).
- Remove `MCP` section fake (`context7 • engram` / `○ disconnected`) (REQ-3).
- Remove `LSP` section fake (REQ-3).
- Colors: replace `headerStyle` literal `#569cd6` with `m.styles.Accent` (or
  `theme.Accent`); replace sidebar `Background("#1a1a1a")` with `theme.Background`
  (OpenCode `#0a0a0a`). Confirm tokens exist in `theme/opencode.go`.

### `views/model_list.go`
- `AvailableModels()`: drop `mimo-v2-free`, `mimo-v2.5` (REQ-4). Keep only real,
  widely-known IDs for any residual fallback.
- `AvailableModelsFiltered()`: when `!anyConfigured`, return `nil` (empty) instead
  of `AvailableModels()` so no fabricated list renders (REQ-4). Caller shows help
  state ("No provider configured — run /login").

### `internal/tui/app.go`
- Verify `SetMCPStatus` / `SetLSPStatus` callers. If only tests call them, keep the
  methods but stop feeding fakes from views. If `app.go` calls them with fake
  values, remove those calls.

### `internal/tui/theme/opencode.go`
- Ensure exported tokens used by views: `Background`, `Accent`, `StatusOK`,
  `StatusError`, `HomeMuted` (in `Styles`). Add if missing.

## Tests (strict TDD)
- RED: add tests asserting fakes are absent (grep-style assertions on `Render()`/
  `View()` output) and that real `tokens`/`cost` flow through.
- GREEN: implement removals.
- Existing `footer_test.go` / `sidebar_test.go` assertions on fake strings MUST be
  updated to assert absence (they currently pin the fakes — that is the bug).

## Risks
- Tests currently encode the fakes; updating them is required, not optional.
- `ctrl+p` binding reality must be confirmed before keeping that hint.
- Theme token names must match `opencode.go` exactly.
