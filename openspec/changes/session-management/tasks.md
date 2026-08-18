# Tasks: Session Management UI

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~365 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | single PR |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Extended SessionMeta + Rename | PR 1 | `go test ./internal/core/... ./internal/adapters/store/...` | N/A — no TUI interaction, pure store logic | `internal/core/session.go`, `internal/adapters/store/session.go` |
| 2 | Undo/Redo + Session List View | PR 1 | `go test ./internal/tui/...` | N/A — Bubbletea Model.Update() tests, no interactive session | `internal/tui/controller.go`, `internal/tui/views/session_list.go` |
| 3 | App wiring + autocomplete | PR 1 | `go test ./internal/tui/...` | N/A — command dispatch only | `internal/tui/app.go`, `internal/tui/autocomplete.go` |

## Phase 1: Enhanced Session Metadata

- [x] 1.1 RED — Test extended `SessionMeta` fields: write `TestSessionMetaExtended` in `internal/core/session_test.go` asserting JSON round-trip with `Name`, `Model`, `Summary`, `UpdatedAt`, `MessageCount`
- [x] 1.2 GREEN — Add fields to `SessionMeta` in `internal/core/session.go`: `Name`, `Model`, `Summary`, `UpdatedAt`, `MessageCount` with `omitempty` JSON tags
- [x] 1.3 RED — Test `NewSessionMeta` populates defaults: write `TestNewSessionMetaDefaults` verifying zero-value fields serialize cleanly

## Phase 2: Session Store Rename

- [x] 2.1 RED — Test `Rename` method: write `TestSessionStoreRename` in `internal/adapters/store/session_test.go` — save session, rename, verify name in loaded session and index
- [x] 2.2 GREEN — Implement `Rename(id, name string) error` on `FileSessionStore` in `internal/adapters/store/session.go`: load, set `Meta.Name`, re-save, update index
- [x] 2.3 RED — Test `Rename` on nonexistent session: write `TestSessionStoreRenameNotFound` verifying error returned
- [x] 2.4 GREEN — Update `Save` in `FileSessionStore` to preserve `UpdatedAt` field when already set

## Phase 3: Undo/Redo in Controller

- [x] 3.1 RED — Test undo stack: write `TestUndoStack` in `internal/tui/controller_test.go` — create controller with messages, push undo point, verify `Undo()` truncates, `Redo()` restores
- [x] 3.2 GREEN — Implement undo/redo in Controller: add `undoSnapshot` type, `undoStack`, `redoStack` slices; add `PushUndo()`, `Undo()`, `Redo()` methods in `internal/tui/controller.go`
- [x] 3.3 RED — Test undo on empty stack: write `TestUndoEmptyStack` verifying no panic, messages unchanged
- [x] 3.4 RED — Test redo clears on new undo point: write `TestRedoClearsOnNewPush` verifying redo stack empties after a new push

## Phase 4: Session List View

- [x] 4.1 RED — Test session list model creation: write `TestSessionListCreate` in `internal/tui/views/session_list_test.go` — create model from `[]SessionMeta`, verify initial state
- [x] 4.2 GREEN — Create `SessionListModel` in `internal/tui/views/session_list.go`: wrap `bubbles/list.Model`, implement `NewSessionListModel`, `Update`, `View`, `Selected()`
- [x] 4.3 RED — Test session list selection: write `TestSessionListSelection` — simulate `tea.KeyDown`, `tea.KeyEnter`, verify selected ID returned
- [x] 4.4 RED — Test session list delete key: write `TestSessionListDelete` — press `d`, verify item removed from list

## Phase 5: App Integration

- [x] 5.1 RED — Test `/sessions` opens interactive list: write `TestAppSessionsInteractive` in `internal/tui/app_test.go` — call `handleSessionsCommand`, verify app enters list mode
- [x] 5.2 GREEN — Wire session list into App: add `sessionList *views.SessionListModel` field, update `handleSessionsCommand` to populate it, update `handleKey` to delegate when list active, add `handleCommand` cases for `/rename`, `/undo`, `/redo`
- [x] 5.3 RED — Test `/undo` dispatches to controller: write `TestAppUndoCommand` — invoke `/undo`, verify controller state changes
- [x] 5.4 GREEN — Update `defaultCommands` in `internal/tui/autocomplete.go`: add `/rename`, `/fork`, `/search`, `/undo`, `/redo`

## Key Learnings
1. SessionMeta currently has only 3 fields — extending it requires updating NewSessionMeta, Save, and the index rebuild path.
2. The controller's messages slice is the undo boundary — snapshots must capture length before mutation.
3. bubbles/list is already in go.mod, so the session list view avoids new dependencies.
4. FileSessionStore.Save already does atomic write — Rename reuses the same Save path.
5. Undo/redo must use independent copies for both save and restore to avoid shared backing array bugs.
6. UpdatedAt should NOT be auto-set by Save — only set explicitly by Rename to avoid breaking CreatedAt-based sorting.
