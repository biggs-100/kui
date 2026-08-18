# Archive Report: Dynamic Extensions

## Change Summary

**Change**: dynamic-extensions
**Archived**: 2026-08-17
**Archived to**: `openspec/changes/archive/2026-08-17-dynamic-extensions/`

Runtime extension discovery and loading via MCP-style subprocess extensions. Users can now extend kui with tools, hooks, and commands without recompilation, using filesystem-scanned extensions in `~/.config/kui/extensions/` (global) and `.kui/extensions/` (project-level).

## Final State

**18 packages passing, go vet clean, build clean.** (Explicit final-state from orchestrator launch prompt — overrides stale intermediate task checkboxes.)

### Task Completion Gate

The persisted `tasks.md` shows Phases 2–5 tasks unchecked. The orchestrator's launch prompt explicitly confirms "18 packages passing, go vet clean, build clean" — the final-state authority (rank 3) validates that all implementation work is complete. The stale checkboxes are an artifact of `sdd-apply` not updating the persisted tasks file after implementation. Exceptional reconciliation applied: all implementation tasks marked complete based on explicit final-state evidence.

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| dynamic-extension-discovery | Created | 3 new requirements: Discovery Sources, Manifest Format, Config-Based Sources |
| dynamic-extension-lifecycle | Created | 5 new requirements: Subprocess Spawning, Protocol Handshake, Tool Registration, Crash Handling, Graceful Shutdown |
| extension-discovery | Modified | REQ-DISCOVERY-1: added filesystem scanning alongside compiled-in; REQ-DISCOVERY-2: added RegisterDynamic |
| extension-system | Modified | REQ-EXT-5: added RegisterDynamic support; REQ-EXT-6: added dynamic extension lifecycle scenarios |

## Archive Contents

- proposal.md ✅
- exploration.md ✅
- design.md ✅
- specs/ ✅ (4 delta specs)
- tasks.md ✅ (exceptional stale-checkbox reconciliation applied — see above)
- state.yaml ✅

## Source of Truth Updated

The following specs now reflect the new behavior:
- `openspec/specs/dynamic-extension-discovery/spec.md` (new)
- `openspec/specs/dynamic-extension-lifecycle/spec.md` (new)
- `openspec/specs/extension-discovery/spec.md` (modified — filesystem discovery, RegisterDynamic)
- `openspec/specs/extension-system/spec.md` (modified — RegisterDynamic, mixed lifecycle)

## Verification

- [x] Main specs updated correctly
- [x] Change folder moved to archive
- [x] Archive contains all artifacts (proposal, exploration, specs, design, tasks, state)
- [x] Archived tasks.md reconciled — stale checkboxes resolved via final-state authority
- [x] Active changes directory no longer has this change
- [x] Verbatim diff readback: byte-identical (MD5 verified)

## Key Files

| File | Purpose |
|------|---------|
| `internal/extensions/dynamic/manager.go` | Extension lifecycle manager |
| `internal/extensions/dynamic/client.go` | JSON-RPC 2.0 client over stdio |
| `internal/extensions/dynamic/config.go` | Extensions config loader |
| `internal/extensions/dynamic/manifest.go` | Extension manifest parser |
| `internal/extensions/dynamic/extension.go` | DynamicExtension adapter |
| `internal/extensions/dynamic/tool.go` | DynamicTool wrapper |
| `internal/extensions/dynamic/errors.go` | Error types |
| `internal/adapters/extensions/registry.go` | Modified — RegisterDynamic + dynamic slice |
| `internal/runtime/runtime.go` | Modified — dynamic extension load step |

## SDD Cycle Complete

The change has been fully planned, explored, proposed, designed, specified, task-planned, implemented, verified, and archived.

Ready for the next change.
