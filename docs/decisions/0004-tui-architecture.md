# ADR-0004: TUI architecture — boundary, observer, concurrency, run model

- Status: accepted
- Date: 2026-08-16

## Context

kui needs an interactive terminal UI for ongoing conversations. The existing
hexagonal core (ADR-0001) enforces stdlib-only imports in `internal/core`,
so third-party UI frameworks cannot cross into the domain. The loop runs
`agent.Run` per prompt on its own goroutine, and events must cross from the
agent goroutine into the Bubble Tea event loop safely. Profile switching
must work mid-session without restarting. The UI must be testable without
a real terminal.

## Decision

### D1: Boundary confinement

Bubble Tea and lipgloss are confined to `internal/tui` and its `views`
subpackage. The core remains stdlib-only; a guard test
(`TestCoreImportsStdlibOnly`) fails the build if any bubbletea import
leaks into `internal/core`. The TUI package owns rendering; core owns
business logic.

### D2: Observer port

A nil-safe `Observer` interface in `internal/core/observer.go` exposes
`OnTurnStart`, `OnTurnEnd`, `OnToolCall`, and `OnToolResult`. The loop
emits events through a recover-wrapped `emitObserver` helper. When the
observer is nil (CLI one-shot mode), the helper is a no-op — zero cost,
zero risk to existing tests.

### D3: Channel + Cmd concurrency

Events cross from the agent goroutine into the Bubble Tea program through
a buffered `chan tea.Msg` (capacity 64). The controller sends events via
`select { case ch <- ev: default: }` — dropping on full or quit, never
blocking the agent loop. The Bubble Tea program receives events through a
re-armed `tea.Cmd` that reads from the channel, preserving the Cmd pipeline
semantics.

### D4: Per-prompt run model

Each prompt submission spawns one goroutine running `agent.Run`. The
controller tracks the active run and blocks further submissions while a
run is in progress. The shared `agent.Agent` persists across prompts,
preserving conversation history. This matches the one-shot `agent.Run`
semantics and avoids interleaving.

### D5: View independence

Views are tested via `Model.Update` and `View()` methods — no Bubble Tea
test harness required for unit tests. Golden files (`-update`) verify
rendering at fixed terminal sizes. The `teatest` package is used only for
integration tests (interactive TAB/quit scenarios).

### D6: Profile switching

TAB and Shift+TAB cycle through `loader.Discover()` profiles with wrap
(`active = ((active ± delta) % n + n) % n`). The controller enqueues a
steering switch (`PendingMessage{SwitchProfile}`) that applies between
turns. History is preserved across switches.

### D7: Three-region layout

The terminal is divided into three horizontal regions: header (profile
tabs), chat (messages + input), and tool (live tool-call list). The layout
adapts to terminal resize. The header shows profile tabs with the active
one highlighted; the chat region displays the message history and input
buffer; the tool region shows tool calls and results from the observer.

## Alternatives considered

- **Allow Bubble Tea in core**: breaks the hexagonal boundary; makes the
  loop untestable without a terminal; violates the guard test.
- **`program.Send` for event delivery**: simpler but bypasses the Cmd
  pipeline and is harder to unit-test in isolation.
- **Session-loop run model**: a long-lived goroutine would need complex
  shutdown and cancellation; per-prompt matches existing `agent.Run`
  semantics.
- **Full streaming (SSE)**: no SSE port exists in core; deferred to a
  later change. Today the controller emits the whole answer as one chunk.

## Consequences

- The core remains stdlib-only and testable with in-memory fakes.
- The observer is nil-safe, so reverting to CLI-only is a clean diff.
- The channel+Cmd pattern is the standard Bubble Tea cross-goroutine
  handoff — familiar to Bubble Tea developers.
- Profile switching works mid-session without restarting.
- Views are unit-testable without a terminal via `Model.Update` + `View()`.
- The three-region layout provides a clean separation of concerns in the
  UI.

## Verification notes

- Guard test: `go test ./internal/core -run TestCoreImportsStdlibOnly`
  confirms no bubbletea in core.
- Controller tests: `go test ./internal/tui/... -race` covers cycle wrap,
  channel overflow, nil observer, and per-prompt model chain.
- View golden tests: `go test ./internal/tui/views/... -update` verifies
  rendering at fixed sizes.
- Integration: `go test ./... -race` — all 11 packages green.
