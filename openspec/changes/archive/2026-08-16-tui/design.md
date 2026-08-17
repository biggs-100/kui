# Design: OpenCode-Style TUI with In-Session Profile Switching

## Technical Approach

Add an OpenCode-style Bubble Tea chat to kui. `internal/tui` owns Bubble Tea + lipgloss; a controller wires views to the existing runtime (store, loader, manager, skills, agent, openai client) without touching core. Core gains one nil-safe stdlib `Observer` port (REQ-LOOP-7) that feeds tool/turn events to the tool view. `agent.Run` executes on its own goroutine per prompt; events cross into the TUI through a buffered channel pumped by a re-armed `tea.Cmd` (REQ-TUI-APP-3). TAB/shift+TAB cycle `loader.Discover()` with wrap and enqueue a steering switch that applies between turns (REQ-TUI-PROF-2/3); each prompt captures `{profile, model}` at submission via the REQ-CLI-4 chain.

## Architecture Decisions

| # | Decision | Options | Tradeoff | Choice |
|---|----------|---------|----------|--------|
| D1 | UI deps boundary | Confine to `internal/tui` vs. allow anywhere | Third-party in core breaks hexagon | Confine; existing `TestCoreImportsStdlibOnly` guard fails on any bubbletea in core (REQ-TUI-APP-4) |
| D2 | Core observer | stdlib `Observer` port vs. callback struct | Interface keeps core decoupled; nil-safe field keeps existing tests untouched | Add `Observer` field + recover-wrapped `emit` helper (REQ-LOOP-7) |
| D3 | Event delivery | `program.Send` vs. channel + `tea.Cmd` | Send is simpler but bypasses Cmd pipeline and is harder to unit-test | Buffered `chan tea.Msg` + one re-armed `cmd: func() tea.Msg { return <-ch }` returned again after each event — the mandated channel+Cmd handoff |
| D4 | Run model | One goroutine per prompt vs. session loop | Per-prompt matches one-shot `agent.Run` semantics; single active run avoids interleaving | One goroutine per prompt, submissions blocked while running |
| D5 | Profile cycle | Resolve name vs. index | Index gives deterministic wrap and rapid-press stepping | `active = ((active ± delta) % n + n) % n`, exactly one step per press |
| D6 | Input widget | `textarea` sub-model vs. plain buffer | Sub-model complicates Update bubbling and golden tests | Plain string buffer in App.Update |
| D7 | Streaming | Provider SSE vs. view chunk-ready | No SSE port exists; view-level chunks keep core untouched | View handles `streamChunkMsg`; today controller emits the whole answer as one chunk then done (SSE deferred) |

D4/D5/D6 keep logic in the controller/app where it is unit-testable without Bubble Tea internals.

## Data Flow

    User ──key events──▶ App.Update ──SubmitPrompt──▶ Controller ──Run goroutine──▶ agent.Run
      ▲                      │                            │  │                          │
      │                      │ ◀──tea.Cmd pump──events chan│  │◀──observer writes───────┘
      └──── tea.Render ◀─────┘      (single cross-goroutine handoff)

- Controller goroutine only sends to `events` (buffered 64; send uses `select { case ch<-ev: default: }` — drop on full/quit, never blocks the loop).
- TAB: App.Update → `SwitchProfile(±1)` → header re-renders + `Steering().Enqueue(PendingMessage{SwitchProfile})`; loop applies it between turns, history preserved.
- Prompt: App resolves model via `agent.ResolveModel` → `client.SetModel` → spawns run; answer/chunks arrive as events.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/core/observer.go` | Create | `Observer` interface + `emit` recover helper |
| `internal/core/loop.go` | Modify | `Observer` field; emit OnTurnStart/End, OnToolCall/Result (nil-safe) |
| `internal/tui/run.go` | Create | `Run(ctx, wiring)` composition; builds store/loader/manager/agent/controller |
| `internal/tui/app.go` | Create | Root `tea.Model`: Init/Update/View, keybindings (q, ctrl+c, tab, shift+tab), three-region layout, resize |
| `internal/tui/controller.go` | Create | Runtime wiring, event channel, cycle logic, `Start/SubmitPrompt/SwitchProfile/Events` |
| `internal/tui/views/chat.go` | Create | Messages, input, chunk rendering, error state |
| `internal/tui/views/header.go` | Create | Profile tab rendering + no-profiles hint |
| `internal/tui/views/tool.go` | Create | Live tool-call/result list (empty on nil observer) |
| `internal/agent/model.go` | Create | Shared `ResolveModel(mem, loader, name)` (moved chain from main.go) |
| `cmd/kui/main.go` | Modify | `kui tui` dispatch (validates client first); one-shot path unchanged |
| `go.mod` | Modify | `bubbletea`, `lipgloss`, `teatest` (test) |
| `docs/decisions/0004-tui-architecture.md` | Create | ADR: boundary, observer, channel+Cmd concurrency |

## Interfaces / Contracts

```go
// core/observer.go — stdlib only
type Observer interface {
    OnTurnStart(); OnTurnEnd()
    OnToolCall(call ToolCall)
    OnToolResult(callID, result string)
}

// internal/tui/controller.go
type Controller struct{ /* wiring, profiles, active, events chan tea.Msg */ }
func (c *Controller) Start(ctx context.Context)
func (c *Controller) SubmitPrompt(text string)      // resolves {profile, model}, spawns run
func (c *Controller) SwitchProfile(delta int)       // wrap cycle + steering enqueue
func (c *Controller) Events() <-chan tea.Msg        // testable event sink

// App messages
type streamChunkMsg struct{ msgID int; delta string }
type streamDoneMsg  struct{ msgID int; err error }  // err → error state (REQ-TUI-CHAT-2)
type toolCallMsg / toolResultMsg / turnMsg
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit (controller) | Cycle wrap/rapid presses, switch enqueued, per-prompt model chain, events emitted | Table-driven; fake `Runner` (Run+Steering) + fake `ModelMemory`; assert on `Events()`; `-race` |
| Unit (app/view) | `Model.Update` with `chunkMsg`, `toolMsg`, tab/shift+tab, resize, quit; empty input; stream error; nil observer | Direct `Model.Update` + `tea.Msg` |
| Golden | Header/tool rendering at fixed size | Golden files via `-update`, deterministic |
| Integration | Interactive TAB cycle + quit | `teatest.NewTestModel`; skip in `-short` |
| Guard | Core stays stdlib; CLI one-shot unchanged | Existing `TestCoreImportsStdlibOnly` + existing CLI tests pass unmodified |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary introduced. `kui tui` is static argument dispatch (no user-supplied command composition); the `bash` tool already exists and is unchanged; observer/UI handoff is in-process goroutine+channel (concurrency, covered above, not a shell boundary). The git/PR rows do not apply.

## Migration / Rollout

No migration required. Rollback: drop `kui tui` dispatch; remove `internal/tui`; observer field is nil-safe so reverting core is clean.

## Open Questions

- [ ] Provider-level SSE streaming (true chunk streaming) deferred — confirm `streamChunkMsg` single-chunk delivery is acceptable for verify.
- [ ] Input disabled vs. queued while a run is active — default: blocked with a running indicator.
