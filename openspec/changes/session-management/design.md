# Design: Session Management UI

## Technical Approach

Extend the existing hexagonal session infrastructure with richer metadata, interactive TUI views, and in-memory undo/redo. Follow the established view pattern (`Model` + `Render()`) for new components. The session list view uses `charmbracelet/bubbles/list` for interactive selection. Undo/redo is a per-session in-memory stack on the Controller — not persisted.

## Architecture Decisions

| Decision | Options | Tradeoff | Choice |
|----------|---------|----------|--------|
| Session list rendering | Custom Bubble Tea model vs `bubbles/list` | Custom = more control, more code; `bubbles/list` = built-in filtering, pagination, keybindings | `bubbles/list` — already in go.mod, reduces TUI code |
| Undo storage | Per-message slice copies vs index-based pointers | Copies = O(n) memory per undo; pointers = O(1) but stale on session mutation | Index-based: store `len(messages)` at each undo point; undo truncates, redo restores from snapshot |
| Rename persistence | Add `Name` to `SessionMeta` vs separate file | Separate file = extra I/O, no atomicity with session; meta field = one write, consistent | Add `Name` to `SessionMeta` — consistent with existing index pattern |
| Search implementation | Linear scan vs FTS index | FTS = complex for MVP, fast at scale; linear = simple, adequate for small JSON files | Linear scan — sessions are small, can add FTS later |
| Fork copy semantics | Deep copy messages vs shared reference | Shared = mutation risk; deep copy = safe, minimal cost for small sessions | Deep copy `[]core.Message` slice |
| Session name resolution | Fallback chain: `Name` → first user message → ID | Always works, no empty states | Name → Summary → ID |

## Data Flow

```
User types /sessions
    → App.handleSessionsCommand()
    → Ctrl.SessionStore().List()
    → []SessionMeta returned
    → App renders SessionListView (bubbles/list)
    → User selects session → Enter
    → App.handleResumeCommand(id)
    → Ctrl.LoadSession(id) → store.Load(id)
    → Chat.LoadHistory(msgs)

User types /undo
    → App.handleUndoCommand()
    → Ctrl.Undo()
    → Ctrl truncates messages to last snapshot point
    → Chat.LoadHistory(msgs) — re-render

User types /fork [id] [index]
    → App.handleForkCommand(args)
    → Ctrl.ForkSession(id, index)
    → store.Load(sourceID) → copy messages[:index]
    → store.Save(newSession with new ID)
    → Ctrl.SetSessionID(newID) → switch active session
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/core/session.go` | Modify | Add `Model`, `UpdatedAt`, `MessageCount`, `Summary`, `Name` fields to `SessionMeta` |
| `internal/adapters/store/session.go` | Modify | Add `Rename(id, name string) error` to port; implement in `FileSessionStore`; update `Save` to populate new meta fields; update sort to use `UpdatedAt` |
| `internal/tui/controller.go` | Modify | Add `undoStack []undoSnapshot`, `redoStack []undoSnapshot` types; add `Undo()`, `Redo()`, `ForkSession()`, `RenameSession()` methods |
| `internal/tui/app.go` | Modify | Add `/rename`, `/fork`, `/search`, `/undo`, `/redo` command cases in `handleCommand`; update `/sessions` to open interactive list; update `handleKey` to delegate to list view when active |
| `internal/tui/views/session_list.go` | Create | `SessionListModel` wrapping `bubbles/list.Model`; methods: `NewSessionListModel`, `Update`, `View`, `Selected()` |
| `internal/tui/session_search.go` | Create | `SearchSessions(query string, store SessionStore) []SearchResult` — linear scan with snippet extraction |
| `internal/tui/autocomplete.go` | Modify | Add `/rename`, `/fork`, `/search`, `/undo`, `/redo` to `defaultCommands` |
| `internal/core/session_test.go` | Modify | Add tests for extended `SessionMeta` fields and `Name` resolution |
| `internal/adapters/store/session_test.go` | Create | Tests for `Rename`, index updates on save with new fields |
| `internal/tui/views/session_list_test.go` | Create | Tests for list model creation, selection, key handling |
| `internal/tui/controller_test.go` | Create | Tests for `Undo`, `Redo`, `ForkSession`, `RenameSession` |

## Interfaces / Contracts

```go
// internal/core/session.go — extended SessionMeta
type SessionMeta struct {
    ID           string `json:"id"`
    Profile      string `json:"profile"`
    Model        string `json:"model,omitempty"`
    Name         string `json:"name,omitempty"`        // custom user-given name
    Summary      string `json:"summary,omitempty"`     // first user message, truncated
    CreatedAt    string `json:"created_at"`
    UpdatedAt    string `json:"updated_at,omitempty"`
    MessageCount int    `json:"message_count,omitempty"`
}

// internal/adapters/store/session.go — extended port
type SessionStore interface {
    Save(session *Session) error
    Load(id string) (*Session, error)
    List() ([]SessionMeta, error)
    Delete(id string) error
    Rename(id string, name string) error  // NEW
}

// internal/tui/controller.go — undo snapshot
type undoSnapshot struct {
    messages []core.Message
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Extended `SessionMeta` JSON round-trip with new fields | `go test ./internal/core/...` — extend existing `TestMessageJSONRoundTrip` |
| Unit | `FileSessionStore.Rename` updates index + file | `go test ./internal/adapters/store/...` — temp dir, create session, rename, verify |
| Unit | Controller `Undo`/`Redo` stack behavior | `go test ./internal/tui/...` — create controller with messages, undo, redo, verify state |
| Unit | Controller `ForkSession` copies messages correctly | `go test ./internal/tui/...` — fork from index, verify new session has correct messages |
| Unit | `SessionListModel` renders and selects | `go test ./internal/tui/views/...` — create model, simulate key presses, verify selection |
| Unit | `SearchSessions` finds matching content | `go test ./internal/tui/...` — mock store with known sessions, search, verify results |
| Unit | Autocomplete includes new commands | `go test ./internal/tui/...` — verify `defaultCommands` contains new entries |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No data migration required. Existing sessions remain valid — new `SessionMeta` fields are optional with zero-value defaults. JSON files with old format deserialize cleanly; new fields populate on next `Save`. The index rebuild path (`rebuildIndex`) already handles partial metadata.

## Open Questions

- [ ] Should `/sessions` interactive list replace the text dump entirely, or show text dump first then offer "press Enter for interactive view"?
- [ ] Should fork from another session switch the active session to the forked one, or keep the current session active?
