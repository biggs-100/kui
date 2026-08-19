# Proposal: Session Persistence

## Intent

kui has zero session persistence. Conversation history lives only in memory during a single `agent.Run()` call and is lost on exit. Users cannot resume work across runs, list past sessions, or reference prior conversations. This is the foundation for all session-aware features.

## Scope

### In Scope
- Session file format: `.kui/sessions/{uuid}.json` with `{id, profile, model, provider, created_at, updated_at, messages}`
- Metadata index: `.kui/sessions/index.json` for fast listing without reading all files
- Core port: `SessionStore` interface in core (save, load, list, delete)
- Store adapter: implement `SessionStore` using JSON files under `.kui/sessions/`
- Agent integration: `Agent.Run()` loads history before execution, returns final history for saving
- TUI save/restore: save on exit, restore on startup (if `--resume` flag or last session)
- TUI commands: `/sessions` (list), `/resume <id>` (switch session)
- CLI subcommands: `kui session list`, `kui session resume <id>`
- Core `Message` and `ToolCall` structs: add JSON tags for disk serialization

### Out of Scope
- Conversation compaction (summarize old turns) — future phase
- Session search/query by content
- Multi-user session isolation
- `--resume` flag on `kui tui` (covered by CLI subcommands for MVP)

## Capabilities

### New Capabilities
- `session-store`: Session persistence adapter — JSON file read/write, index maintenance, CRUD operations on session files
- `session-management`: Session lifecycle — save on exit, restore on start, list/resume commands (TUI and CLI)

### Modified Capabilities
None — this is greenfield. No existing specs change.

## Approach

Follow the exploration recommendation: JSON files + metadata index, matching the established `store.Store` pattern.

1. Add JSON tags to `core.Message` and `core.ToolCall` (stdlib only, no new imports)
2. Define `SessionStore` port in core with `Save`, `Load`, `List`, `Delete` methods
3. Implement `FileSessionStore` adapter in `internal/adapters/store/` — reads/writes `.kui/sessions/{uuid}.json`, maintains `index.json`
4. Wire session store into `agent.Agent` — load history on Run, expose final history for save
5. Wire into `tui.Controller` — save on quit, restore on startup, handle `/sessions` and `/resume`
6. Add `kui session list` and `kui session resume <id>` CLI subcommands
7. Atomic writes: write to temp file, then rename (existing pattern in store)

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/core/provider.go` | Modified | Add JSON tags to Message and ToolCall |
| `internal/core/session.go` | New | SessionStore port definition |
| `internal/adapters/store/session.go` | New | FileSessionStore implementation |
| `internal/adapters/store/session_test.go` | New | Tests for session store |
| `internal/agent/agent.go` | Modified | Accept/return session history |
| `internal/tui/controller.go` | Modified | Session lifecycle management |
| `internal/tui/app.go` | Modified | `/sessions` and `/resume` commands |
| `cmd/kui/main.go` | Modified | `kui session list/resume` subcommands |
| `cmd/kui/flags.go` | Modified | Resume flag on Options |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Large session files from long conversations | Medium | Future compaction; MVP has no limit |
| Concurrent writes from multiple kui instances | Low | UUID session IDs + atomic temp+rename |
| Core gains JSON tags (stdlib-only) | Low | JSON tags are stdlib, no new imports |
| Index drift from session files | Low | Reconcile on startup (read files, rebuild index) |

## Rollback Plan

1. Remove `openspec/changes/session-persistence/` directory
2. Delete `internal/core/session.go`, `internal/adapters/store/session.go`, `internal/adapters/store/session_test.go`
3. Revert changes to `provider.go`, `agent.go`, `controller.go`, `app.go`, `main.go`, `flags.go`
4. No data migration needed — `.kui/sessions/` directory is simply not created if feature is removed

## Dependencies

- None external. All within kui's existing Go module and store pattern.

## Success Criteria

- [ ] `kui session list` shows all saved sessions with profile, model, timestamp, message count
- [ ] `kui session resume <id>` restores a session and injects history into the agent
- [ ] TUI `/sessions` command lists sessions inline
- [ ] TUI `/resume <id>` switches to a different session
- [ ] Session files are valid JSON and can be manually inspected
- [ ] Existing tests continue to pass (no regressions)
- [ ] New session store tests achieve ≥80% coverage

## Proposal question round

Before finalizing, I need your input on these product decisions:

1. **Auto-save timing**: Should kui save the session after every prompt response, or only on explicit `/save` or exit? Auto-save is safer (no data loss on crash) but writes more frequently.

2. **Session identity**: UUID-based IDs are opaque. Should `kui session list` show a truncated UUID (e.g., `550e8400`), or should we generate human-friendly labels (e.g., `coder-2026-08-18-1015`)?

3. **Session limits**: Should MVP enforce a max message count or file size per session, or leave it unbounded? Long tool-use conversations can grow large.

4. **History injection scope**: When resuming a session, should the full message history be sent to the provider on the first prompt, or only a summary/context window? Full history preserves continuity but costs tokens.

5. **New session creation**: Should `kui tui` always start a fresh session (current behavior), or should it auto-resume the last session if one exists?
