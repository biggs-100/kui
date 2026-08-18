# Proposal: Session Management UI

## Intent

kui has basic session persistence — save, restore, list, and resume via `/sessions` and `/resume <id>`. But the UX is minimal: `/sessions` dumps raw text into the chat status bar, and there's no way to rename, fork, search, or undo within a session. Users need a proper session management layer to navigate and control their conversation history without leaving the TUI.

## Current Gap

The existing session infrastructure is solid at the persistence layer:

- `core.SessionStore` port with `Save`, `Load`, `List`, `Delete`
- `FileSessionStore` adapter with JSON files + metadata index
- Controller auto-saves after each response and on quit
- `/sessions` lists sessions as raw text in the status bar
- `/resume <id>` restores a session by ID

What's missing:

| Capability | Today | Target |
|------------|-------|--------|
| Session list | Text dump in status bar | Interactive TUI view with selection |
| Session metadata | ID, profile, created_at only | + model, message count, last modified |
| Rename | Not possible | `/rename <new-name>` or interactive |
| Fork | Not possible | Branch from a specific message into new session |
| Search | Not possible | Full-text search across session messages |
| Undo/redo | Not possible | Step back/forward through conversation turns |

## Proposed Solution

### 1. Enhanced Session Metadata

Extend `SessionMeta` to carry richer metadata for display and search:

```go
type SessionMeta struct {
    ID           string `json:"id"`
    Profile      string `json:"profile"`
    Model        string `json:"model"`
    CreatedAt    string `json:"created_at"`
    UpdatedAt    string `json:"updated_at"`
    MessageCount int    `json:"message_count"`
    Summary      string `json:"summary,omitempty"` // first user message, truncated
}
```

### 2. Interactive Session List (`/sessions`)

Replace the text dump with an interactive Bubble Tea list view:

- Arrow keys to navigate, Enter to resume, `d` to delete, `r` to rename, `f` to fork
- Shows: name (truncated summary or custom name), profile, model, date, message count
- Sorted by most recently modified
- `/sessions` still works as a command; the list view is the enhanced path

### 3. Session Rename (`/rename`)

- `/rename <new-name>` — renames the active session
- Interactive rename in the session list view (press `r`)
- Stores custom name in `SessionMeta` alongside the auto-generated ID
- Display uses custom name when set, falls back to summary or ID

### 4. Session Fork (`/fork`)

- `/fork` — creates a new session from the current conversation up to the last message
- `/fork <session-id> <message-index>` — branches from a specific point in another session
- Forked session gets a new ID but copies messages from the fork point
- Original session is untouched

### 5. Session Search (`/search`)

- `/search <query>` — full-text search across all session messages
- Returns matching sessions with highlighted matching snippets
- Results are navigable; Enter to resume a matched session
- Searches `content` field of all `Message` objects in loaded sessions

### 6. Undo/Redo Within Session

- `/undo` — removes the last assistant response and its preceding user message
- `/redo` — restores the last undone pair (if available)
- Implemented as an undo stack on the controller's `messages` slice
- Undo/redo state is per-session and not persisted (session file reflects current state)

## Scope

### In Scope

- Extended `SessionMeta` with model, updated_at, message_count, summary
- Enhanced `/sessions` with interactive list view (Bubble Tea list component)
- `/rename <name>` command and interactive rename
- `/fork` command (from current session and from specific point)
- `/search <query>` command with result navigation
- `/undo` and `/redo` commands
- Controller undo stack (in-memory, per-session)
- Session name resolution (custom name → summary → ID fallback)
- TUI autocomplete entries for new commands

### Out of Scope

- Session merge (combining two sessions)
- Conversation compaction / summarization
- Cross-session copy/paste
- Session tagging or folders
- Persistent undo (undo stack resets on session resume)
- Drag-and-drop session reordering

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/core/session.go` | Modified | Extend `SessionMeta` with model, updated_at, message_count, summary fields |
| `internal/adapters/store/session.go` | Modified | Update index format, add `Rename` method to `SessionStore` port |
| `internal/tui/controller.go` | Modified | Add undo stack, undo/redo logic, fork support |
| `internal/tui/app.go` | Modified | Add `/rename`, `/fork`, `/search`, `/undo`, `/redo` command handlers |
| `internal/tui/views/session_list.go` | New | Interactive session list Bubble Tea model |
| `internal/tui/session_search.go` | New | Search logic across loaded sessions |
| `internal/tui/autocomplete.go` | Modified | Add new command entries |
| `openspec/specs/session-management/` | New | Delta specs for each new capability |

## Success Criteria

- [ ] `/sessions` opens an interactive list view showing: name, profile, model, date, message count
- [ ] Arrow keys navigate the list; Enter resumes; `d` deletes; `r` renames
- [ ] `/rename <name>` renames the active session; custom name persists across sessions
- [ ] `/fork` creates a new session from current conversation
- [ ] `/fork <id> <index>` branches from a specific point in another session
- [ ] `/search <query>` returns sessions matching the query with context snippets
- [ ] `/undo` removes the last assistant response; `/redo` restores it
- [ ] New commands appear in autocomplete suggestions
- [ ] All existing session persistence tests continue to pass
- [ ] New features have unit tests at ≥80% coverage

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Interactive list adds TUI complexity | Medium | Medium | Follow existing Bubble Tea patterns (header, tool views); keep list as a single model |
| Fork from arbitrary message index is fragile | Medium | Low | Validate index bounds; fallback to fork-from-end on invalid index |
| Undo stack memory growth on long sessions | Low | Low | Stack stores message slices by reference, not copies; bounded by session length |
| Search across large sessions is slow | Medium | Low | Linear scan for MVP; sessions are small JSON files; can add FTS later |
| Rename collides with ID-based resume | Low | Medium | Display shows custom name, resume still accepts ID; ID is never overwritten |

## Rollback Plan

1. Remove `openspec/changes/session-management/` directory
2. Revert `internal/core/session.go` changes (remove new fields from `SessionMeta`)
3. Revert `internal/adapters/store/session.go` changes (remove `Rename` from port)
4. Delete `internal/tui/views/session_list.go`, `internal/tui/session_search.go`
5. Revert `internal/tui/controller.go` changes (remove undo stack)
6. Revert `internal/tui/app.go` changes (remove new command handlers)
7. No data migration needed — existing sessions remain valid; new fields are optional with defaults

## Dependencies

- Existing `session-persistence` change (already implemented)
- Bubble Tea list component (charmbracelet/bubbles — already in go.mod for other views)
- No new external dependencies required
