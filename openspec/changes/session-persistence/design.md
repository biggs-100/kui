# Design: Session Persistence

## Technical Approach

Add JSON-file session persistence following the established `store.Store` pattern. A new `SessionStore` port in core defines the contract; a `FileSessionStore` adapter implements it under `.kui/sessions/`. The agent accepts pre-loaded history on `Run()` and exposes final history for saving. The TUI controller manages auto-save after each response and on quit. CLI subcommands and TUI slash commands provide listing and resume.

## Architecture Decisions

### Decision: Storage format

| Option | Tradeoff | Decision |
|--------|----------|----------|
| SQLite | Fast queries, single file, new dependency | Rejected — adds `modernc.org/sqlite`, heavier than needed for MVP |
| JSON files only | Simple, no deps, inspectable; slow listing at scale | Rejected — reading all files for listing is wasteful |
| **JSON files + metadata index** | Fast listing via `index.json`, no new deps, reconcilable | **Chosen** — matches store pattern, index can be rebuilt on drift |

### Decision: Session ID format

| Option | Tradeoff | Decision |
|--------|----------|----------|
| UUID | Globally unique, opaque | Rejected — poor UX in `kui session list` |
| **Human-friendly** (`profile-YYYY-MM-DD-HHMM`) | Readable, collision risk at same-minute granularity | **Chosen** — append 4-char hex suffix if needed for collisions |

### Decision: History injection scope

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Full history | Perfect continuity, high token cost | **Chosen for MVP** — compaction is a future phase |
| Summary window | Lower cost, loses context | Deferred — out of scope per proposal |

## Data Flow

```
User prompt
    │
    ▼
Controller.SubmitPrompt()
    │
    ├─▶ Agent.Run(ctx, prompt, history)
    │       │
    │       ▼
    │   core.Agent.Run() ──▶ Provider.Chat()
    │       │
    │       ▼
    │   Final messages[] returned
    │
    ├─▶ FileSessionStore.Save(session)
    │       │
    │       ▼
    │   .kui/sessions/{id}.json  (atomic write)
    │   .kui/sessions/index.json (updated)
    │
    ▼
Stream chunks → ChatModel → View
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/core/provider.go` | Modify | Add `json:"..."` tags to `Message` and `ToolCall` structs |
| `internal/core/session.go` | Create | `SessionStore` port: `Save`, `Load`, `List`, `Delete` + `Session`/`SessionMeta` types |
| `internal/adapters/store/session.go` | Create | `FileSessionStore` — JSON files + index under `.kui/sessions/` |
| `internal/adapters/store/session_test.go` | Create | Unit tests: CRUD, atomic writes, index rebuild |
| `internal/agent/agent.go` | Modify | `Run()` accepts `[]core.Message` history, returns `[]core.Message` final |
| `internal/tui/controller.go` | Modify | Accept `SessionStore`, auto-save after response, save on quit, `/sessions`/`/resume` |
| `internal/tui/app.go` | Modify | Route `/sessions` and `/resume <id>` to controller |
| `internal/tui/views/chat.go` | Modify | `LoadHistory([]core.Message)` to restore messages for rendering |
| `cmd/kui/main.go` | Modify | `kui session list` and `kui session resume <id>` subcommands |
| `cmd/kui/flags.go` | Modify | Add `Resume` field to `Options` |

## Interfaces / Contracts

```go
// internal/core/session.go
type Session struct {
    ID        string           `json:"id"`
    Profile   string           `json:"profile"`
    Model     string           `json:"model"`
    Provider  string           `json:"provider"`
    CreatedAt time.Time        `json:"created_at"`
    UpdatedAt time.Time        `json:"updated_at"`
    Messages  []Message        `json:"messages"`
}

type SessionMeta struct {
    ID           string    `json:"id"`
    Profile      string    `json:"profile"`
    Model        string    `json:"model"`
    Provider     string    `json:"provider"`
    CreatedAt    time.Time `json:"created_at"`
    MessageCount int       `json:"message_count"`
}

type SessionStore interface {
    Save(session Session) error
    Load(id string) (Session, error)
    List() ([]SessionMeta, error)
    Delete(id string) error
}
```

Agent signature change:
```go
// Before:
func (a *Agent) Run(ctx context.Context, prompt string) (string, error)

// After:
func (a *Agent) Run(ctx context.Context, prompt string, history []core.Message) (string, []core.Message, error)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `FileSessionStore` CRUD, atomic writes, index rebuild | `go test` with temp dirs via `KUI_HOME` |
| Unit | `core.Message` JSON round-trip | Marshal/unmarshal assertions |
| Unit | Agent history injection | Mock provider, verify messages sent |
| Integration | TUI auto-save after response | `teatest` — send prompt, verify session file |
| Integration | `/sessions` and `/resume` commands | `teatest` — type command, verify output |
| Integration | CLI `kui session list/resume` | `go test` — invoke `run()` with args, check stdout |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. `.kui/sessions/` is created on first save. Existing kui installs have no session data. The `--no-session` flag remains a no-op placeholder.

## Open Questions

- [ ] Should the agent's `Run()` signature change break the existing `Runner` interface, or should history be a separate method? (Design assumes signature change — cleaner but wider impact.)
- [ ] Index reconciliation: rebuild from files on every `List()` call, or only when index is missing/corrupted? (Design assumes on-drift only for performance.)
