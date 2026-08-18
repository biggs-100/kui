# Exploration: Session Persistence

## Current State

kui has **zero session persistence** today. The conversation history lives exclusively in memory during a single `agent.Run()` call and is lost when the process exits. Key evidence:

- **`core.Agent.Run()`** (loop.go:60): builds `messages := []Message{{Role: RoleUser, Content: prompt}}` fresh every call — no history is loaded from disk, no history is saved after completion.
- **`agent.Agent`** (agent.go): owns `steering` and `followUp` queues but no history storage.
- **`tui.Controller.SubmitPrompt()`** (controller.go:133): spawns a goroutine per prompt via `runner.Run(ctx, text)` — each prompt is independent; the controller has no message history across prompts.
- **`tui.ChatModel`** (views/chat.go): holds `messages []Message` in memory for rendering only — never serialized to disk.
- **`store.Store`** (adapters/store/store.go): persists only `models.json` (per-profile model overrides) and `active` (active profile name). No conversation data.
- **`--no-session`** flag (flags.go:28): explicitly a no-op placeholder per REQ-CLI-21.
- **`runtime.Runtime`** (runtime.go): builds and reloads runtime state but never touches conversation history.

The `core.Message` type (provider.go:28-33) is the serialization-ready unit:

```go
type Message struct {
    Role       string
    Content    string
    ToolCall   *ToolCall
    ToolCallID string
}
```

This is already JSON-serializable and is the natural persistence boundary.

## Affected Areas

- **`internal/adapters/store/`** — must gain session read/write methods alongside existing `models.json`/`active` files
- **`internal/core/loop.go`** — `Agent.Run()` must accept pre-loaded history and return final history for saving
- **`internal/core/provider.go`** — `Message` type needs JSON tags for disk serialization
- **`internal/agent/agent.go`** — `Agent` must expose history access for save/restore
- **`internal/tui/controller.go`** — must manage session lifecycle (save on quit, restore on start, session switching)
- **`internal/tui/app.go`** — slash commands `/sessions`, `/resume` need handling
- **`internal/tui/run.go`** — wiring must pass session store to controller
- **`internal/tui/views/chat.go`** — must support loading historical messages for rendering
- **`cmd/kui/main.go`** — CLI needs `kui session list`, `kui session resume <id>` subcommands, and `--resume` flag
- **`cmd/kui/flags.go`** — `Options` needs `Resume`, `SessionID` fields
- **`internal/runtime/runtime.go`** — `Build()` must optionally load session history

## Approaches

### Approach 1: JSON files in `.kui/sessions/`

Each session is a JSON file: `.kui/sessions/{uuid}.json` containing `{id, profile, model, provider, created_at, updated_at, messages: []Message}`.

- **Pros**: Simple, no new dependencies, matches kui's JSON-based store pattern (`models.json`), easy to inspect/debug, atomic writes (write-to-temp + rename)
- **Cons**: Listing sessions requires reading all files (slow at scale), no query capability, file fragmentation
- **Effort**: Low

### Approach 2: SQLite database in `.kui/sessions.db`

Single SQLite file with `sessions` and `messages` tables. Uses `modernc.org/sqlite` (pure Go, no CGO).

- **Pros**: Fast listing/sorting, indexed queries, atomic transactions, conversation compaction is natural (DELETE + summarize), single file
- **Cons**: New dependency (SQLite driver), more complex than JSON files, harder to debug raw data, potential CGO issues on some platforms (mitigated by pure-Go driver)
- **Effort**: Medium

### Approach 3: JSON files + metadata index

JSON files for session data (Approach 1) plus a lightweight `sessions/index.json` that caches metadata (id, profile, model, timestamp, message count) for fast listing without reading every file.

- **Pros**: Fast listing, no new dependencies, inspectable files, incremental — can start with Approach 1 and add index later
- **Cons**: Index can drift from files (needs reconciliation on corruption), two files to keep in sync
- **Effort**: Low-Medium

## Recommendation

**Approach 3: JSON files + metadata index**, implemented in two phases:

1. **Phase 1 (MVP)**: JSON files in `.kui/sessions/` — session save/restore, `kui session list` reads directory, `kui session resume <id>` loads and injects history. This matches kui's existing simplicity and the store adapter pattern.

2. **Phase 2 (polish)**: Add `sessions/index.json` for fast listing, conversation compaction for old sessions, and `--resume` flag on `kui tui`.

This mirrors Pi's session model (JSON-based persistence under a config directory) while staying consistent with kui's hexagonal architecture: the session store is an adapter implementing a `SessionStore` port in core.

### Storage format (recommended)

```
.kui/
├── models.json          (existing)
├── active               (existing)
└── sessions/
    ├── index.json       (metadata cache)
    ├── {uuid-1}.json    (full session)
    └── {uuid-2}.json    (full session)
```

Session JSON structure:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "profile": "coder",
  "model": "gpt-4o",
  "provider": "openai",
  "created_at": "2026-08-18T10:00:00Z",
  "updated_at": "2026-08-18T10:15:00Z",
  "messages": [
    {"role": "user", "content": "hello"},
    {"role": "assistant", "content": "Hi! How can I help?"},
    {"role": "system", "content": "Profile switched to coder..."}
  ]
}
```

## Risks

1. **History size**: Long conversations with tool calls can produce large JSON files. Mitigation: conversation compaction (summarize old turns) as a future enhancement.
2. **Concurrent access**: Two kui instances writing to the same session could corrupt data. Mitigation: UUID-based session IDs + atomic file writes (temp + rename); only the owning instance should write.
3. **Core boundary**: `core.Message` needs JSON tags but core must stay stdlib-only. Mitigation: JSON tags are stdlib (`encoding/json`), no new imports needed.
4. **Migration**: No existing sessions to migrate — this is greenfield.
5. **TUI chat view**: `views.ChatModel` stores `Message` (UI type), not `core.Message` (domain type). Need a conversion layer or unified type. Mitigation: the controller already translates between them; add a restore path that feeds `AppendMessage` on startup.

## Ready for Proposal

Yes. The exploration is complete. The orchestrator should proceed to **sdd-propose** with the recommended approach (JSON files + metadata index, phased implementation).

## Key Learnings

1. kui has zero session persistence today — `--no-session` is explicitly a no-op placeholder per REQ-CLI-21.
2. `core.Message` is already JSON-serializable and is the natural persistence boundary for conversation history.
3. The `store.Store` adapter pattern (JSON files under `.kui/`) is the established persistence convention — session files should follow it.
4. The TUI controller spawns independent goroutines per prompt with no shared history — session restore requires feeding historical messages into `ChatModel.AppendMessage` on startup.
5. The hexagonal boundary means the session store must be an adapter implementing a new `SessionStore` port in core, keeping core stdlib-only.
