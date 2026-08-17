# tui-app Specification

## Purpose

`kui tui` is the interactive primary workflow: a Bubble Tea application that composes the profile-tab header, the chat message view, and the live tool view. It owns the UI dependencies and never runs UI work on the agent loop's goroutine.

## Requirements

### Requirement: REQ-TUI-APP-1 — Entrypoint & Lifecycle

`kui tui` MUST start the Bubble Tea program composing the header, chat, and tool views. The app MUST quit on `q` or `ctrl+c`. If startup fails (e.g. invalid provider configuration), the app MUST NOT render; the CLI MUST exit non-zero with an actionable stderr message.

#### Scenario: Starts the full layout

- GIVEN a valid provider configuration and at least one profile
- WHEN `kui tui` starts
- THEN a Bubble Tea program renders the header, chat, and tool views

#### Scenario: Quit on ctrl+c

- GIVEN a running TUI
- WHEN the user presses `ctrl+c`
- THEN the app exits cleanly with status zero

#### Scenario: Startup failure

- GIVEN invalid provider configuration
- WHEN `kui tui` starts
- THEN no TUI renders
- AND the exit status is non-zero with an actionable stderr message

### Requirement: REQ-TUI-APP-2 — Layout & Resize

The app MUST render three regions — header (profile tabs), chat (messages and input), and tool view. A window resize MUST reflow all three regions and MUST NOT crash the app.

#### Scenario: Three-region layout

- GIVEN a default-size terminal
- WHEN the app renders
- THEN the header, chat, and tool regions are all visible

#### Scenario: Resize reflows

- GIVEN a running TUI
- WHEN the window resizes
- THEN all three regions reflow to the new dimensions
- AND the app keeps running

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
