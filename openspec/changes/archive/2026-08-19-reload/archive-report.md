# Archive Report: reload

**Date**: 2026-08-19
**Status**: Complete (intentional stale-checkbox reconciliation)
**Mode**: openspec

## Summary

Hot reload command for kui. Provides `/reload` slash command that cancels active runs, tears down MCP/extensions, re-reads configurable state, rebuilds runtime, and swaps only on clean build. Includes manager race fix, agent setters, runtime lifecycle package, and TUI integration.

## Artifacts

- proposal.md ✅
- design.md ✅
- specs/ ✅ (5 domains: agent-setters, extension-wiring, reload-concurrency, runtime-lifecycle, tui-reload)
- tasks.md ✅ (18/18 tasks complete)

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| agent-setters | Created | New domain — SetSkills, SetProvider, SetHooks on Agent |
| extension-wiring | Created | New domain — extAPI tool+hook registration |
| reload-concurrency | Created | New domain — manager mutex, concurrent reload safety |
| runtime-lifecycle | Created | New domain — Build, Reload, Close in internal/runtime |
| tui-reload | Created | New domain — /reload command, status rendering, cancel-and-wait |

## Stale Checkbox Reconciliation

Phase 2 (6 tasks) and Phase 3 (7 tasks) had stale unchecked checkboxes. Reconciled based on orchestrator's explicit instruction that all work is implemented and pushed. The tasks were completed during apply but the checkboxes were not updated in the persisted artifact.

## Key Files

- `internal/runtime/runtime.go` — Build/Reload/Close composition
- `internal/runtime/extapi.go` — ExtensionAPI implementation
- `internal/tui/controller.go` — reload lifecycle, cancel-and-wait
- `internal/tui/app.go` — /reload slash command dispatch
