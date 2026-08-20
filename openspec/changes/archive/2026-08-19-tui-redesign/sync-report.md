# Sync Report: tui-redesign

**Status**: synced
**Date**: 2025-08-19
**Change**: tui-redesign — OpenCode-style TUI

---

## Executive Summary

Delta specs from `openspec/changes/tui-redesign/specs/` have been synced into canonical `openspec/specs/`. Two domains were processed: `tui-app` (MODIFIED + ADDED) and `tui-home` (new canonical). No destructive operations, no same-domain collisions, no blockers.

---

## Domains Synced

### tui-app

**Canonical file**: `openspec/specs/tui-app/spec.md`

| Operation | Requirement | Notes |
|---|---|---|
| MODIFIED | REQ-TUI-APP-1 — Entrypoint & Lifecycle | Updated to describe home→session flow; preserved original "Startup failure" scenario from canonical (delta requirement text still covers it) |
| MODIFIED | REQ-TUI-APP-2 — Layout & Resize | Replaced three-region layout with dual home/session layout model |
| ADDED | REQ-TUI-APP-5 — Route System | New requirement for home/session route state |
| ADDED | REQ-TUI-APP-6 — Footer Variants | New requirement for route-dependent footer rendering |
| ADDED | REQ-TUI-APP-7 — Theme "opencode" | New requirement for opencode theme color palette |

Preserved unchanged canonical requirements: REQ-TUI-APP-3 (Concurrency Boundary), REQ-TUI-APP-4 (Dependency Boundary).

### tui-home

**Canonical file**: `openspec/specs/tui-home/spec.md` (created new)

| Operation | Requirement | Notes |
|---|---|---|
| NEW | REQ-TUI-HOME-1 — Centered Layout | Full new spec |
| NEW | REQ-TUI-HOME-2 — ASCII Logo | Full new spec |
| NEW | REQ-TUI-HOME-3 — Bordered Prompt | Full new spec |
| NEW | REQ-TUI-HOME-4 — Minimal Footer | Full new spec |
| NEW | REQ-TUI-HOME-5 — Prompt Submission | Full new spec |
| NEW | REQ-TUI-HOME-6 — Keyboard Shortcuts | Full new spec |

No pre-existing canonical `tui-home/spec.md` existed. Entire delta spec became canonical.

---

## Merge Decisions

1. **REQ-TUI-APP-1 "Startup failure" scenario preserved**: The delta's requirement text still includes "If startup fails, the app MUST NOT render; the CLI MUST exit non-zero with an actionable stderr message." The original canonical had a dedicated scenario for this behavior that was not listed in the delta's scenario block. To avoid spec coverage regression, the "Startup failure" scenario was merged into the synced requirement alongside the three new scenarios from the delta.

2. **No REMOVED requirements**: Neither delta spec contained `## REMOVED Requirements` sections.

3. **No RENAMED requirements**: Neither delta spec contained `## RENAMED Requirements` sections.

4. **No destructive sync**: No large MODIFIED blocks or REMOVED requirements requiring explicit approval.

---

## Active Same-Domain Collisions

None. `tui-redesign` is the only active change in `openspec/changes/`.

---

## Validation

- Verified no other active changes touch `openspec/specs/tui-app/spec.md` or `openspec/specs/tui-home/spec.md`
- Verified delta specs contain no RENAMED requirements
- Verified delta specs contain no REMOVED requirements
- Verified MODIFIED requirements reference existing canonical requirement names
- Verified canonical tui-app/spec.md preserves REQ-TUI-APP-3 and REQ-TUI-APP-4 unchanged
- Applied `rules.sync` from `openspec/config.yaml`: no sync-specific rules defined (only general `specs` rules for Given/When/Then and RFC 2119 keywords, which the delta specs already follow)

---

## Next Recommended Phase

`sdd-archive` — verification is CONDITIONAL PASS (all implementation tasks complete, all programmatic checks pass; remaining items are parent-owned manual/visual verification tasks).

---

## Structured Status

- artifactStore: openspec
- state: synced
- domains: [tui-app, tui-home]
- canonicalFilesUpdated: [openspec/specs/tui-app/spec.md, openspec/specs/tui-home/spec.md]
- addedRequirements: [REQ-TUI-APP-5, REQ-TUI-APP-6, REQ-TUI-APP-7, REQ-TUI-HOME-1, REQ-TUI-HOME-2, REQ-TUI-HOME-3, REQ-TUI-HOME-4, REQ-TUI-HOME-5, REQ-TUI-HOME-6]
- modifiedRequirements: [REQ-TUI-APP-1, REQ-TUI-APP-2]
- removedRequirements: []
- sameDomainCollisions: []
- destructiveOperations: false
