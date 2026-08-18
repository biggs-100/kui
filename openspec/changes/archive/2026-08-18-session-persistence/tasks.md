# Tasks: Session Persistence

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 450–550 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Core types + SessionStore port + JSON tags | PR 1 | `go test ./internal/core/...` | N/A — pure types, no I/O | `internal/core/session.go`, `provider.go` JSON tags |
| 2 | FileSessionStore adapter + unit tests | PR 2 | `go test ./internal/adapters/store/...` | temp dirs via `KUI_HOME` | `internal/adapters/store/session.go`, `session_test.go` |
| 3 | Agent + TUI + CLI integration | PR 3 | `go test ./...` | `kui session list`, `kui session resume <id>` | `agent.go`, `controller.go`, `app.go`, `main.go`, `flags.go`, `views/chat.go` |

## Phase 1: Core Types & Port

- [x] 1.1 RED: Write `internal/core/session_test.go` — `TestMessageJSONRoundTrip`: marshal `Message` with `ToolCall`, unmarshal, assert fields match
- [x] 1.2 GREEN: Add `json:"..."` tags to `core.Message` and `core.ToolCall` in `internal/core/provider.go`
- [x] 1.3 RED: Write `internal/core/session_test.go` — `TestSessionStoreInterface`: compile-time assertion that `*FileSessionStore` satisfies `SessionStore`
- [x] 1.4 GREEN: Create `internal/core/session.go` — `Session`, `SessionMeta`, `SessionStore` interface (`Save`, `Load`, `List`, `Delete`)
- [x] 1.5 REFACTOR: Verify `go vet ./internal/core/...` and `go test ./internal/core/...` pass

## Phase 2: FileSessionStore Adapter

- [x] 2.1 RED: Write `internal/adapters/store/session_test.go` — `TestSaveCreatesFile`: save session, assert `.kui/sessions/{id}.json` exists
- [x] 2.2 GREEN: Create `internal/adapters/store/session.go` — `FileSessionStore` struct, `NewSessionStore(root)`, `Save()` with atomic write (temp + rename)
- [x] 2.3 RED: `TestSaveUpdatesIndex`: save session, assert `index.json` contains metadata entry
- [x] 2.4 GREEN: Implement `updateIndex()` helper — read/merge/write `index.json`
- [x] 2.5 RED: `TestLoadReturnsFullSession`: save then load, assert messages and metadata match
- [x] 2.6 GREEN: Implement `Load(id)` — read JSON, unmarshal to `Session`
- [x] 2.7 RED: `TestListReturnsMetadata`: save 3 sessions, `List()` returns 3 `SessionMeta` entries sorted by `created_at` desc
- [x] 2.8 GREEN: Implement `List()` — read index file, return metadata slice
- [x] 2.9 RED: `TestDeleteRemovesFileAndIndex`: save then delete, assert file and index entry gone
- [x] 2.10 GREEN: Implement `Delete(id)` — remove file, update index
- [x] 2.11 RED: `TestIndexRebuiltOnDrift`: delete index file, call `List()`, assert index rebuilt from session files
- [x] 2.12 GREEN: Add index rebuild logic in `List()` when index is missing/corrupted
- [x] 2.13 RED: `TestHumanFriendlyID`: call `GenerateSessionID("coder")`, assert format `coder-YYYY-MM-DD-HHMM`
- [x] 2.14 GREEN: Implement `GenerateSessionID(profile)` — timestamp-based with 4-char hex suffix for collisions
- [x] 2.15 REFACTOR: Verify `go vet ./internal/adapters/store/...` and `go test ./internal/adapters/store/...` pass

## Phase 3: Agent History Integration

- [x] 3.1 RED: Write `internal/agent/agent_test.go` — `TestRunAcceptsHistory`: mock provider, verify `[]core.Message` history is prepended to provider call
- [x] 3.2 GREEN: Change `agent.Agent.Run()` signature to `Run(ctx, prompt, history []core.Message) (string, []core.Message, error)` — prepend history to messages, return final `[]core.Message`
- [x] 3.3 RED: `TestRunReturnsFinalMessages`: mock provider with tool call, assert returned messages include user prompt, assistant response, tool result
- [x] 3.4 GREEN: Ensure `Run()` returns accumulated `messages` slice (not just last content)
- [x] 3.5 Update `internal/tui/run.go` `agentRunner.Run()` to match new signature — pass empty history, discard returned messages
- [x] 3.6 Update `internal/tui/controller.go` `Runner` interface: `Run(ctx, prompt string) (string, error)` → `Run(ctx, prompt string, history []core.Message) (string, []core.Message, error)`
- [x] 3.7 Update `SubmitPrompt()` to pass session history to runner, capture returned messages
- [x] 3.8 REFACTOR: Verify `go vet ./...` and `go test ./internal/core/... ./internal/agent/... ./internal/tui/...` pass

## Phase 4: TUI Session Lifecycle

- [x] 4.1 Add `sessionStore core.SessionStore` and `sessionID string` fields to `Controller` struct
- [x] 4.2 Add `SetSessionStore(store)` and `SetSessionID(id)` methods to Controller
- [x] 4.3 In `SubmitPrompt()`, after run completes: call `sessionStore.Save()` with updated messages (auto-save after response)
- [x] 4.4 Add `SaveSession()` method — save current session to store, update index
- [x] 4.5 Add `LoadSession(id)` method — load session, return `[]core.Message` for history injection
- [x] 4.6 In `app.go` `handleKey()`: intercept Ctrl+C and `/quit`/`/exit` — call `ctrl.SaveSession()` before `tea.Quit`
- [x] 4.7 Add `/sessions` command handler in `app.go` — call `sessionStore.List()`, render formatted table to chat view
- [x] 4.8 Add `/resume <id>` command handler in `app.go` — call `ctrl.LoadSession(id)`, inject history, update session ID
- [x] 4.9 Add `LoadHistory([]views.Message)` method to `views.ChatModel` — populate messages from session for rendering
- [x] 4.10 Update `internal/tui/run.go` `Run()` to create `FileSessionStore`, wire into controller
- [x] 4.11 REFACTOR: Verify `go test ./internal/tui/...` pass

## Phase 5: CLI Subcommands

- [ ] 5.1 Add `Resume string` field to `Options` in `cmd/kui/flags.go`
- [ ] 5.2 Add `resume` to `stringFlags` map in `cmd/kui/flags.go`
- [ ] 5.3 Add `kui session` subcommand dispatcher in `cmd/kui/main.go`
- [ ] 5.4 Implement `runSessionList()` — create `FileSessionStore`, call `List()`, print table to stdout
- [ ] 5.5 Implement `runSessionResume(id)` — load session, validate exists, launch TUI with history injected
- [ ] 5.6 Update `runTUI()` to accept optional resume ID — load session history if provided
- [ ] 5.7 Add session usage text constant in `cmd/kui/main.go`
- [ ] 5.8 REFACTOR: Verify `go test ./cmd/kui/...` and full `go test ./...` pass

## Phase 6: Integration Verification

- [ ] 6.1 Run `go build ./...` — verify clean build
- [ ] 6.2 Run `go vet ./...` — verify no vet issues
- [ ] 6.3 Run `go test ./...` — all tests pass, no regressions
- [ ] 6.4 Manual test: `kui session list` shows "No sessions found" on fresh install
- [ ] 6.5 Manual test: Start TUI, send prompt, quit → session saved to `.kui/sessions/`
- [ ] 6.6 Manual test: `kui session list` shows saved session with correct metadata
- [ ] 6.7 Manual test: `kui session resume <id>` restores session with history
