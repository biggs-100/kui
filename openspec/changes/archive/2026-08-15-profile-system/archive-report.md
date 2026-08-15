# Archive Report: profile-system

**Archived**: 2026-08-15
**From**: `openspec/changes/profile-system/` (active)
**To**: `openspec/changes/archive/2026-08-15-profile-system/`
**Branch**: `feat/profile-system/5-cli-model`
**Commits** (chained 1→2→3→4→5): `c04dd56` (core: two-level loop + steering/follow-up queues), `8de92ab` (permissions: rulesets + tool hiding), `daa8b73` (profile: layered loader + hot switch), `ed88762` (agent: on-demand skills, `.kui` store, agent wrapper), `09e1756` (CLI: profile subcommands + per-profile model resolution)

## Closure Summary

The change is fully implemented and verified at close. Terminal facts (highest-ranked sources):

- **Verdict**: `pass_with_warnings` (per `verify-report.md`, evidence_revision `sha256:26aaa8b5564610de09652ff5ba9024d528d827456716dd0b69f9f98d34402cd1`)
- **CRITICAL findings**: 0 (blockers: 0 — archive gate satisfied)
- **Requirements**: 23/23 compliant
- **Scenarios**: 52/52 compliant with passing runtime covering tests
- **Tests**: 149 passed / 0 failed / 2 environment-conditional skips (Windows symlink tests in `internal/adapters/tools`) — `go test -count=1 ./...` exit 0; `go build ./...`, `go vet ./...`, `golangci-lint run ./...` clean; `-race` clean on core/agent/adapters; gofmt clean on changed files (LF-normalized)
- **Tasks**: 30/30 complete (phases 1–6, all `[x]` in persisted `tasks.md` — no stale unchecked tasks; archive proceeded without checkbox reconciliation)

## Specs Synced (Source of Truth)

Five new domains had no existing main spec, so each delta spec WAS a full spec and was copied mechanically (`cp` → `diff -r` → `mv`, byte-identity verified). Two domains had existing main specs and received a content merge (delta applied with model-authored edit, preserving all non-delta requirements):

| Domain | Action | Details | Path |
|--------|--------|---------|------|
| profile-runtime | Created (full spec) | 5 requirements (REQ-PROFILE-1..5) | `openspec/specs/profile-runtime/spec.md` |
| profile-permissions | Created (full spec) | 4 requirements (REQ-PERM-1..4) | `openspec/specs/profile-permissions/spec.md` |
| profile-skills | Created (full spec) | 3 requirements (REQ-SKILL-1..3) | `openspec/specs/profile-skills/spec.md` |
| profile-cli | Created (full spec) | 3 requirements (REQ-PCLI-1..3) | `openspec/specs/profile-cli/spec.md` |
| steering-followup | Created (full spec) | 3 requirements (REQ-QUEUE-1..3) | `openspec/specs/steering-followup/spec.md` |
| agent-loop | Updated (merge) | MODIFIED REQ-LOOP-1 (two-level loop, scenarios preserved + 2 added); ADDED REQ-LOOP-5, REQ-LOOP-6; REQ-LOOP-2..4 preserved intact | `openspec/specs/agent-loop/spec.md` |
| agent-cli | Updated (merge) | ADDED REQ-CLI-3, REQ-CLI-4; REQ-CLI-1..2 preserved intact | `openspec/specs/agent-cli/spec.md` |

The agent-loop merge preserved the full REQ-LOOP-1..4 block per the spec skill's lost-content warning: REQ-LOOP-1 was replaced with the delta's complete MODIFIED requirement (which itself carries the preserved direct-answer and multi-step scenarios plus the new steering/termination scenarios), and REQ-LOOP-2/3/4 were untouched.

All 5 synced full-spec files verified byte-identical to the archived delta specs (`diff -r` exit 0, empty output).

## Final-State Facts (forwarded by orchestrator, outrank intermediate snapshots)

1. **verify-report WARNINGs flagged, NOT fixed — carried as follow-ups**: (1) the skills index rides the steering queue as a `RoleUser` message instead of the system prompt (REQ-SKILL-3 wording deviation — observable scenario assertions still pass at the wrapper level); (2) the first provider request carries neither SYSTEM.md nor the skills index — profile context shapes request 2 onward. Both are wiring nuances that break no scenario; a future core hook would fix them. Not resolved at close; recorded as open follow-ups.
2. **yaml.v3 v3.0.1 is the module's first external dependency** — adapter-only (`internal/adapters/profile`), pinned; the core guard test enforces stdlib-only.
3. **gofmt repo-wide CRLF noise on Windows (core.autocrlf) is pre-existing**; changed files verified LF-clean.
4. **PR 1 of the chain is ALREADY OPEN as #10** (issue #9 approved) — no pending PR creation for slice 1. Slices 2–5 PRs come after archive.

## Snapshot-Derived Claims (attributed, not terminal facts)

- Coverage figures per `verify-report` at verification time: core 89.8%, agent 94.0%, permissions 100.0%, profile 90.0%, openai 86.4%, skills 84.5%, store 72.9%, tools 86.5%, cmd/kui 0% (exec-binary smoke artifact; coverage does not cross the process boundary — expected).
- Additional verify-report warning (retained, non-blocking): apply-progress TDD-evidence completeness — the retained apply-progress observation carries per-task RED/GREEN rows only for the final batch (PR 5); phases 1–4 marked complete with commit refs but without per-task rows in the retained revision. Independently verified: all referenced test files exist and pass, so this is an evidence-retention gap, not a protocol failure.
- Suggestions carried (informational): cmd/kui 0% coverage is a subprocess-test attribution artifact; consider seeding SYSTEM.md + skills index into the first provider request at session start to match the design data flow.

## Intentional-Warnings Markers

None — archive proceeded under ordinary repository policy. No intentional partial archive, no stale-checkbox reconciliation, no destructive merge (all spec updates additive; the agent-loop merge replaced only the requirement named in the delta and added two new ones — no large sections removed).

## Gates

- **Task Completion Gate**: passed — 30/30 checked in persisted `tasks.md` (archived copy re-verified: 30 `[x]`, 0 `[ ]`), no reconciliation needed.
- **CRITICAL gate**: passed — 0 CRITICAL findings, 0 blockers in verify-report.
- **Native Review Receipt Gate**: `reviewGate` structurally absent from the launch (no structured status block, no review artifacts for this candidate) — archive proceeded under ordinary repository policy.
- **Action Context Guard**: no `workspace-planning` mode signaled; operations confined to `openspec/`.

## Mechanical Copy Verification

- 5 spec syncs (full specs): `diff -r` exit 0, empty output each (source delta vs. temp copy before rename).
- 2 spec merges: delta requirements applied by content merge into the existing main specs (model-authored edit by design — these are merges, not byte copies); verified by re-read that non-delta requirements were preserved.
- Archive move: recursive snapshot before move; `diff -r` snapshot vs. archived folder exit 0 (DIFF_EXIT=0), empty output; source directory confirmed gone. `archive-report.md` is additive (written after the readback).
- Archive contents verified present: `proposal.md`, `design.md`, `tasks.md`, `verify-report.md`, `_meta.yaml`, `specs/{agent-cli,agent-loop,profile-cli,profile-permissions,profile-runtime,profile-skills,steering-followup}/spec.md`.
- Active `openspec/changes/` no longer contains this change (only `archive/` remains).

## Audit Trail

Archived folder is immutable (audit trail). Source-of-truth main specs under `openspec/specs/` now reflect the shipped behavior. Engram copy of this report: topic key `sdd/profile-system/archive-report`, observation ID `18348` (sync `obs-e0c4fa93059b0ce4`), `capture_prompt: false`.
