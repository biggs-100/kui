# tui-app Specification

## Purpose

`kui tui` is the interactive primary workflow: a Bubble Tea application that composes the profile-tab header, the chat message view, and the live tool view. It owns the UI dependencies and never runs UI work on the agent loop's goroutine.

## Requirements

### Requirement: REQ-TUI-APP-1 — Entrypoint & Lifecycle

`kui tui` MUST start the Bubble Tea program. On first launch, it MUST render the home screen (centered logo + prompt). After prompt submission, it MUST render the session view (chat + tool view). The app MUST quit on `q` or `ctrl+c`. If startup fails, the app MUST NOT render; the CLI MUST exit non-zero with an actionable stderr message.

(Previously: Always renders header, chat, and tool views directly)

#### Scenario: First launch shows home

- GIVEN a valid provider configuration
- WHEN `kui tui` starts
- THEN the home screen renders with centered logo and prompt

#### Scenario: After prompt, shows chat

- GIVEN the home screen is visible
- WHEN the user submits a prompt
- THEN the session view renders with chat and tool views

#### Scenario: Quit on ctrl+c

- GIVEN a running TUI (home or session)
- WHEN the user presses `ctrl+c`
- THEN the app exits cleanly with status zero

#### Scenario: Startup failure

- GIVEN invalid provider configuration
- WHEN `kui tui` starts
- THEN no TUI renders
- AND the exit status is non-zero with an actionable stderr message

### Requirement: REQ-TUI-APP-2 — Layout & Resize

The app MUST support two layouts: home (flex-spacer centered logo + prompt + footer) and session (header + chat + tool view + footer with sidebar). `wide` MUST be `width > 120` (Previously: `width >= 110`). Sidebar width MUST be `42` (Previously: `30`). Session `contentWidth` MUST be `width - (sidebarVisible?42:0) - 4`. When `!wide` sidebar MUST render as overlay with backdrop `RGBA(0,0,0,70)`. A window resize MUST reflow current layout and MUST NOT crash.
(Previously: sidebar 30@110, no contentWidth calc)

#### Scenario: Wide shows sidebar inline

- GIVEN width 130
- WHEN session renders
- THEN sidebar 42 cols is visible inline and contentWidth is 84

#### Scenario: Narrow overlays sidebar

- GIVEN width 100
- WHEN session renders
- THEN contentWidth is 96 and sidebar renders as overlay with backdrop

#### Scenario: Resize reflows

- GIVEN running TUI width 120
- WHEN resized to 160
- THEN layout reflows without panic

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

### Requirement: REQ-TUI-APP-5 — Route System

The app MUST maintain a route state: `home` or `session`. The route MUST switch from `home` to `session` when a prompt is submitted. The route MUST switch from `session` to `home` when the user requests a new session (future: Ctrl+N).

#### Scenario: Initial route is home

- GIVEN `kui tui` starts
- WHEN the app renders for the first time
- THEN the route is `home`

#### Scenario: Prompt submission switches to session

- GIVEN the route is `home`
- WHEN the user submits a prompt
- THEN the route changes to `session`

### Requirement: REQ-TUI-APP-6 — Footer Variants

The app MUST render distinct footers: Home footer is empty/plugin slot (no fabricated `dir • LSP • MCP`). Session footer MUST mirror `routes/session/footer.tsx`: when connected shows `• N LSP` + `⊙ N MCP` + permission `△ N` + `/status`; when welcome (not connected) cycles `Get started /connect` via tick. Counts MUST come from real `sync.data.*` or be omitted as muted, never fabricated.
(Previously: Home `directory • LSP • MCP • /status` minimal, Session `tokens (percent%) • $cost • MCP: N connected • LSP: status`)

#### Scenario: Session connected footer shows dots

- GIVEN LSP 2 connected and MCP 1 failed
- WHEN footer renders
- THEN dump contains `• 2` and `⊙ 1` with status hint `/status`

#### Scenario: Welcome tick cycles

- GIVEN no sync data (not connected)
- WHEN 10s tick fires
- THEN footer cycles `Get started → /connect`

#### Scenario: No fabrication when absent

- GIVEN `sync.data.lsp/mcp` absent
- WHEN footer renders
- THEN counts are omitted as muted, not `0` faked as connected

### Requirement: REQ-TUI-APP-7 — Theme "opencode"

The app MUST include theme "opencode" with 40+ fields matching `assets/opencode.json` and derivation helpers `tint/selectedForeground/generateSyntax`. Theme MUST load from JSON. No hex outside `internal/tui/theme` MAY exist.
(Previously: 25 fields, hardcoded hexes)

#### Scenario: Opencode 40 fields load

- GIVEN `Load("opencode")`
- WHEN inspected
- THEN `backgroundPanel/Element/Menu` and `markdown*/syntax*/diff*` are set

### Requirement: REQ-TUI-APP-8 — Border Primitives and Toast/Title

System MUST provide `ui/border` with `EmptyBorder` and `SplitBorder` (`┃ left, ╹ bottom` vs `│/└` drift must be exact) and decorative bottom `▀` for prompt. It MUST set terminal title to `OpenCode` on home and `OC | {title}` on session. Toast MUST live inside home centered column and session scroll area.

#### Scenario: Chat uses ┃ not │

- GIVEN user message part rendered
- WHEN dump compared at 120 cols
- THEN left border char is `┃` not `│`

#### Scenario: Title reflects route

- GIVEN route home
- WHEN title sequence emitted
- THEN title is `OpenCode`

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
