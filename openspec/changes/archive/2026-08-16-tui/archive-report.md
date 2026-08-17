# Archive Report: tui

**Change**: tui
**Archive Date**: 2026-08-16
**Archived to**: `openspec/changes/archive/2026-08-16-tui/`

## Summary

OpenCode-style Bubble Tea TUI with in-session profile switching. Interactive chat view with TAB-driven profile cycling, live tool output, and per-prompt `{profile, model}` context. Core stays stdlib-only via a guarded dependency boundary.

## Branch & Commits

| PR | Commit | Description |
|----|--------|-------------|
| #19 | d27d7a8 | Observer port + ResolveModel (Slice A) |
| #20 | 443e9ae | Controller — runtime wiring, event delivery, cycle logic (Slice B) |
| #21 | 9adfe58 | Views — header, chat, tool (Slice C) |
| #22 | bfc04c3 | App + CLI entrypoint (Slice D) |
| #23 | ae5645e | ADR + docs cleanup (Slice E) |

**Branch**: feat/tui/5-docs (final)

## Task Completion

- **Total**: 39
- **Complete**: 39
- **Incomplete**: 0
- **All tasks checked**: ✅

## Verification

- **Verdict**: PASS
- **Requirements**: 16/16 covered
- **Scenarios**: 32/32 compliant
- **Tests**: 217 passed, 0 failed, 0 skipped
- **Build**: `go build ./...` — clean
- **Vet**: `go vet ./...` — clean
- **Race detector**: No races (`-race` flag on all packages)
- **Test packages**: 11 (cmd/kui, internal/adapters/*, internal/agent, internal/core, internal/tui, internal/tui/views)

## Design Decisions Followed

| Decision | Description | Followed |
|----------|-------------|----------|
| D1 | UI deps confined to `internal/tui` | ✅ |
| D2 | stdlib Observer + emit helper | ✅ |
| D3 | Channel + `tea.Cmd` handoff | ✅ |
| D4 | One goroutine per prompt | ✅ |
| D5 | Index-based profile cycle | ✅ |
| D6 | Plain input buffer | ✅ |
| D7 | SSE deferred; single-chunk today | ✅ |

## Spec Compliance

### tui-app (8 scenarios) — REQ-TUI-APP-1/2/3/4
- Entrypoint & lifecycle, layout & resize, concurrency boundary, dependency boundary

### tui-chat (6 scenarios) — REQ-TUI-CHAT-1/2/3
- Prompt submission, streaming answer rendering, per-prompt context stability

### tui-profile-switcher (8 scenarios) — REQ-TUI-PROF-1/2/3/4
- Profile tabs, TAB cycle with wrap, session-active switch semantics, no-profiles fallback

### tui-tool-view (3 scenarios) — REQ-TUI-TOOL-1/2
- Live tool events, graceful degradation

### agent-loop (4 scenarios) — REQ-LOOP-7
- Observer port (nil-safe, stdlib)

### agent-cli (3 scenarios) — REQ-CLI-5
- TUI subcommand dispatch

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| tui-app | Created | Full spec — 4 requirements, 8 scenarios |
| tui-chat | Created | Full spec — 3 requirements, 6 scenarios |
| tui-profile-switcher | Created | Full spec — 4 requirements, 8 scenarios |
| tui-tool-view | Created | Full spec — 2 requirements, 3 scenarios |
| agent-cli | Updated | Added REQ-CLI-5 (1 requirement, 3 scenarios) |
| agent-loop | Updated | Added REQ-LOOP-7 (1 requirement, 4 scenarios) |

## Source of Truth Updated

The following main specs now reflect the new behavior:
- `openspec/specs/tui-app/spec.md`
- `openspec/specs/tui-chat/spec.md`
- `openspec/specs/tui-profile-switcher/spec.md`
- `openspec/specs/tui-tool-view/spec.md`
- `openspec/specs/agent-cli/spec.md`
- `openspec/specs/agent-loop/spec.md`

## Key Files Created/Modified

| File | Action |
|------|--------|
| `internal/core/observer.go` | Created — Observer interface + emit helpers |
| `internal/core/loop.go` | Modified — Observer field + event emission |
| `internal/agent/model.go` | Created — ResolveModel extraction |
| `internal/tui/run.go` | Created — Run composition |
| `internal/tui/app.go` | Created — Root tea.Model |
| `internal/tui/controller.go` | Created — Runtime wiring + event delivery |
| `internal/tui/views/header.go` | Created — Profile tab rendering |
| `internal/tui/views/chat.go` | Created — Message view |
| `internal/tui/views/tool.go` | Created — Live tool events |
| `cmd/kui/main.go` | Modified — `kui tui` dispatch |
| `go.mod` | Modified — bubbletea, lipgloss, teatest |
| `docs/decisions/0004-tui-architecture.md` | Created — ADR |

## Issues

- **CRITICAL**: None
- **WARNING**: None
- **SUGGESTION**: 1 — No apply-progress artifact found (RED/GREEN evidence independently verifiable from test files on disk)

## Audit Trail

- `git mv` used for archive move (tracked by git)
- `diff -r` readback: PASS (archive-report excluded as additive)
- All 39 tasks checked in persisted tasks.md
- No CRITICAL issues in verify-report
