# tui-app Specification

## Purpose

`kui tui` is the interactive primary workflow: a Bubble Tea application that renders a single-column transcript (chat + tool view) with a bordered editor and a 2-line dim footer, pi-style. It owns the UI dependencies and never runs UI work on the agent loop's goroutine.

## Requirements

### Requirement: REQ-TUI-APP-1 — Entrypoint & Lifecycle

`kui tui` MUST start the Bubble Tea program and MUST ALWAYS render the transcript view (transcript scroll + bordered editor + 2-line footer). There MUST be NO home route. On fresh start (empty history) it MUST render only the vacant transcript view (bordered editor + 2-line footer); it MUST NOT render an inline changelog (no honest data source exists; fabricating one would violate honesty rules). The app MUST quit on `q` or `ctrl+c`. Startup failure MUST NOT render; the CLI MUST exit non-zero with actionable stderr.
(Previously: first launch rendered home screen, session after submit; then: fresh start rendered inline changelog)

#### Scenario: Fresh start shows vacant transcript only (no changelog)

- GIVEN empty history at 120 cols
- WHEN `kui tui` starts
- THEN transcript, bordered editor and 2-line footer render with no inline changelog

#### Scenario: Startup failure

- GIVEN invalid provider configuration
- WHEN `kui tui` starts
- THEN nothing renders and exit is non-zero with stderr

### Requirement: REQ-TUI-APP-2 — Layout & Resize

The app MUST render a single column: minimal header + transcript scroll + bordered editor + 2-line dim footer. It MUST NOT render a sidebar (inline nor overlay/backdrop), canvas cell painting, viewport height budget, or input `┃` bar/meta-row/fade. Resize MUST reflow without crash; narrow terminals MUST only shrink transcript/editor.
(Previously: home/session layouts, sidebar 42 inline/overlay, contentWidth calc, canvas + viewport budget)

#### Scenario: No sidebar at any width

- GIVEN width 130 with session state
- WHEN View renders
- THEN no 42-col sidebar or backdrop appears

#### Scenario: Narrow survives

- GIVEN width 60
- WHEN View renders
- THEN layout shrinks without panic; footer truncates, never fabricates

### Requirement: REQ-TUI-APP-3 — Concurrency Boundary

`agent.Run` MUST execute on a goroutine separate from the TUI. The loop goroutine MUST NOT mutate UI state directly; all UI updates MUST be dispatched via `tea.Cmd`. No UI work MAY run on the loop's goroutine.

#### Scenario: Turn events reach the UI

- GIVEN a multi-step turn running on the loop goroutine
- WHEN events are produced
- THEN they are delivered to the UI through `tea.Cmd`
- AND UI state is never written from the loop goroutine

### Requirement: REQ-TUI-APP-4 — Dependency Boundary

Bubble Tea and lipgloss imports MUST exist only under `internal/tui`. The core package guard test MUST fail on any Bubble Tea dependency in core, proven by `go list -deps` on the core package.

#### Scenario: Core excludes UI deps

- GIVEN the guard test compiled against the core package
- WHEN `go list -deps` runs on core
- THEN bubbletea and lipgloss do not appear
- AND the guard test passes

#### Scenario: UI import in core blocked

- GIVEN a core file importing bubbletea
- WHEN the guard test runs
- THEN the guard test fails, blocking the change

### Requirement: REQ-TUI-APP-6 — Footer Variants

The footer MUST be exactly 2 dim lines: L1 `cwd (branch) session`; L2 left stats, right `(provider) model` + thinking. Unknown branch/tokens/session MUST be omitted (never fabricated). A spinner accent line MAY appear above the editor while busy; `IdleStatus` MUST reserve 2 lines so the layout never jumps.
(Previously: Home empty/plugin slot; Session `• N LSP + ⊙ N MCP + △ N + /status` with welcome tick)

#### Scenario: Footer contract exact

- GIVEN cwd `/repo`, branch `main`, session `dev`, provider `openai`
- WHEN footer renders
- THEN L1 shows path + `(main)` and L2 right shows `(openai) model`

#### Scenario: Unknowns omitted

- GIVEN unknown branch and no token stats
- WHEN footer renders
- THEN branch/stats are omitted muted, never `0` or invented names

### Requirement: REQ-TUI-APP-7 — Theme "opencode"

The app MUST include theme "opencode" with 40+ fields matching `assets/opencode.json` and derivation helpers `tint/selectedForeground/generateSyntax`. Theme MUST load from JSON. No hex outside `internal/tui/theme` MAY exist.
(Previously: 25 fields, hardcoded hexes)

#### Scenario: Opencode 40 fields load

- GIVEN `Load("opencode")`
- WHEN inspected
- THEN `backgroundPanel/Element/Menu` and `markdown*/syntax*/diff*` are set

### Requirement: REQ-TUI-APP-8 — Border Primitives and Toast/Title

System MUST provide `ui/border` with `EmptyBorder` and `SplitBorder` and MUST set terminal title to `kui` on start and `kui | {title}` with session. Toasts MUST NOT render anywhere (see REQ-TUI-DLG-5).
(Previously: Toast lived inside home centered column and session scroll area)

#### Scenario: No toast renders

- GIVEN any error/status event
- WHEN View dumps
- THEN no floating toast appears; status shows in the transient line

### Requirement: REQ-TUI-APP-9 — Locale and Formatting Invariants

System MUST format numbers via `Intl.NumberFormat`-equivalent `toLocaleString`, money with 2 decimals, timestamps via `Locale.todayTimeOrDateTime`, durations via `formatDuration`. Spinner color MUST be `agent.color`.

#### Scenario: Tokens locale formatted

- GIVEN 1234567 tokens
- WHEN sidebar renders
- THEN dump shows `1,234,567 tokens`

### Requirement: REQ-TUI-APP-10 — Keymap Base/Modal/Leader

System MUST implement keymap stack `base`/`modal` plus leader token. Bindings MUST be formatted via `formatKeyBindings` (leader + aliases `pgup→pgup`). Modal Esc MUST clear filter then close (see REQ-TUI-DLG-4). All bindings MUST be declared in table, not hard-coded scattered handlers.

#### Scenario: Leader binding formats

- GIVEN binding `leader + p`
- WHEN `formatKeyBindings` called
- THEN output contains leader prefix correctly

#### Scenario: Goldens lock layout

- GIVEN app at 80/120/160 widths
- WHEN `View()` dumped
- THEN `testdata/app_*.txt` matches OpenCode column count ±1
