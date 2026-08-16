# Tasks: OpenCode-Style TUI with In-Session Profile Switching

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 500–650 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (A) → PR 2 (B) → PR 3 (C) → PR 4 (D) → PR 5 (E) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| A | Observer port + ResolveModel | PR 1 | `go test ./internal/core/... ./internal/agent/...` | `go test -race` on observer emit; fake Runner in model_test | observer.go + model.go + loop.go diffs removable cleanly |
| B | Controller (event delivery, cycle, submit) | PR 2 | `go test ./internal/tui/...` | fake Runner + fake ModelMemory; `go test -race` | controller.go removable; no views yet |
| C | Views (header, chat, tool) | PR 3 | `go test ./internal/tui/views/...` | golden files via `-update`; nil observer rendering | views/ dir removable; controller unaffected |
| D | App + CLI entrypoint | PR 4 | `go test ./internal/tui/... ./cmd/kui/...` | `teatest.NewTestModel` for interactive TAB+quit | app.go + main.go dispatch removable |
| E | ADR + docs cleanup | PR 5 | `go vet ./...` | N/A (docs only) | docs/ removable |

## Phase 1: Foundation — Observer Port & ResolveModel (Slice A)

- [x] 1.1 RED: `internal/core/observer_test.go` — nil observer no-op; emit with observer receives events; emit contains panic (REQ-LOOP-7 scenarios)
- [x] 1.2 GREEN: `internal/core/observer.go` — create `Observer` interface (OnTurnStart/End, OnToolCall, OnToolResult) + `emit` recover helper (stdlib only)
- [x] 1.3 RED: `internal/agent/model_test.go` — table-driven tests for `ResolveModel(store, loader, name)` covering saved > yaml > env > default chain (REQ-CLI-4)
- [x] 1.4 GREEN: `internal/agent/model.go` — extract `ResolveModel` from `cmd/kui/main.go` (req-cli-4 chain); make it public and reusable
- [x] 1.5 RED: `internal/core/loop_observer_test.go` — loop with observer gets OnTurnStart/End + OnToolCall/Result; loop with nil observer unchanged (REQ-LOOP-7)
- [x] 1.6 GREEN: `internal/core/loop.go` — add `Observer` field; emit events at turn start/end and tool call/result points; nil-safe
- [x] 1.7 Verify: `go test ./internal/core/... ./internal/agent/... -race` — all pass, existing guard test unmodified

## Phase 2: Controller — Runtime Wiring (Slice B)

- [x] 2.1 RED: `internal/tui/controller_test.go` — table-driven: cycle wrap/rapid presses, switch enqueued, per-prompt model chain, events emitted on Run completion; fake Runner (Run+Steering) + fake ModelMemory
- [x] 2.2 GREEN: `internal/tui/controller.go` — `Controller` struct (profiles, active index, events chan tea.Msg, wiring); `Start`, `SubmitPrompt`, `SwitchProfile`, `Events` methods; channel + tea.Cmd handoff (D3)
- [x] 2.3 RED: `internal/tui/controller_test.go` — add nil observer rendering scenario; channel overflow drops on full (D3 select-default)
- [x] 2.4 GREEN: `internal/tui/controller.go` — wire Observer into Run goroutine; implement select-default drop on channel full
- [x] 2.5 Verify: `go test ./internal/tui/... -race -count=1` — controller tests pass

## Phase 3: Views — Header, Chat, Tool (Slice C)

- [x] 3.1 RED: `internal/tui/views/header_test.go` — golden tests: two tabs + active marked; no-profiles hint (REQ-TUI-PROF-1/4)
- [x] 3.2 GREEN: `internal/tui/views/header.go` — `HeaderModel` with profile names, active index; lipgloss tab rendering; no-profiles fallback
- [x] 3.3 RED: `internal/tui/views/chat_test.go` — golden tests: message list grows on chunk; empty input ignored; stream error renders error state (REQ-TUI-CHAT-1/2)
- [x] 3.4 GREEN: `internal/tui/views/chat.go` — `ChatModel` with messages slice, input buffer, `AppendChunk`, `SetError`; message rendering with profile/model metadata (REQ-TUI-CHAT-3)
- [x] 3.5 RED: `internal/tui/views/tool_test.go` — golden tests: tool calls render; nil observer → empty list (REQ-TUI-TOOL-1/2)
- [x] 3.6 GREEN: `internal/tui/views/tool.go` — `ToolModel` with tool events list; append call/result; nil-safe empty state
- [x] 3.7 Verify: `go test ./internal/tui/views/... -update` — all golden tests pass

## Phase 4: App + CLI Entrypoint (Slice D)

- [x] 4.1 RED: `internal/tui/app_test.go` — `Model.Update`: chunkMsg grows answer; tab/shift+tab cycles profile; resize reflows; q/ctrl+c quits; empty input ignored; nil observer (REQ-TUI-APP-1/2/3/4)
- [x] 4.2 GREEN: `internal/tui/app.go` — `App` (tea.Model): Init/Update/View; three-region layout; keybindings; delegates to controller
- [x] 4.3 RED: `internal/tui/run_test.go` — `Run(ctx, wiring)` composes store/loader/manager/controller; startup failure returns error (REQ-TUI-APP-1)
- [x] 4.4 GREEN: `internal/tui/run.go` — `Run(ctx, wiring)` composition; builds store, loader, manager, agent, controller; starts controller goroutine
- [x] 4.5 RED: `cmd/kui/main_test.go` — `kui tui` dispatches; startup validation failure prints stderr; one-shot prompt unchanged (REQ-CLI-5)
- [x] 4.6 GREEN: `cmd/kui/main.go` — add `kui tui` dispatch; validate client first; call `internal/tui.Run`; update usage text
- [x] 4.7 RED: guard test scenario — `TestCoreImportsStdlibOnly` still passes after all changes (bubbletea not in core deps)
- [x] 4.8 Verify: `go test ./... -race -count=1` — full suite green

## Phase 5: Documentation & Cleanup (Slice E)

- [ ] 5.1 Create `docs/decisions/0004-tui-architecture.md` — ADR: boundary confinement (D1), observer port (D2), channel+Cmd concurrency (D3), per-prompt run model (D4)
- [ ] 5.2 Update `cmd/kui/main.go` usage string — add `kui tui` entry (REQ-CLI-5)
- [ ] 5.3 Verify: `go vet ./...` + `go build ./...` — clean
