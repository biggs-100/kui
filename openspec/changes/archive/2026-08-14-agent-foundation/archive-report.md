# Archive Report: agent-foundation

**Archived**: 2026-08-14
**From**: `openspec/changes/agent-foundation/` (active)
**To**: `openspec/changes/archive/2026-08-14-agent-foundation/`
**Branch**: `feat/agent-foundation`
**Commits**: `1210990` (core), `2caf8dd` (tools), `6425ab2` (provider), `c57dba6` (model field), `28c56c8` (gitignore), `27ac26f` (CLI), `32f5ed2` (ADRs)

## Closure Summary

The change is fully implemented and verified at close. Terminal facts (highest-ranked sources):

- **Verdict**: `pass_with_warnings` (per `verify-report.md`, evidence_revision `sha256:77ed2954...`)
- **CRITICAL findings**: 0 (no blockers — archive gate satisfied)
- **Requirements**: 14/14 compliant
- **Scenarios**: 30/30 compliant with passing runtime covering tests
- **Tests**: 55 passing / 0 failing / 2 environment-conditional skips (symlink privilege) — `go test -count=1 ./...` exit 0; `go build ./...`, `go vet ./...`, `golangci-lint run ./...`, `gofmt -l .` all clean
- **Tasks**: 21/21 complete (phases 1–6, all `[x]` in tasks.md — no stale unchecked tasks; archive proceeded without checkbox reconciliation)

## Specs Synced (Source of Truth)

Greenfield: no main spec existed for any domain, so each delta spec WAS a full spec and was copied mechanically (`cp` → `diff -r` → `mv`, byte-identity verified):

| Domain | Action | Path |
|--------|--------|------|
| agent-loop | Created (full spec) | `openspec/specs/agent-loop/spec.md` |
| agent-tools | Created (full spec) | `openspec/specs/agent-tools/spec.md` |
| provider-openai-compatible | Created (full spec) | `openspec/specs/provider-openai-compatible/spec.md` |
| agent-cli | Created (full spec) | `openspec/specs/agent-cli/spec.md` |

All 4 synced files verified byte-identical to the archived delta specs (`diff -r` exit 0, empty output, post-move).

## Final-State Facts (forwarded by orchestrator, outrank intermediate snapshots)

1. **OPENAI_MODEL extension — APPROVED, additive**: commit `c57dba6` added an explicit `model` field to every chat request from `OPENAI_MODEL` (default `gpt-4o-mini`), per explicit user approval. Recorded as an approved additive extension to REQ-PROV-1 scope; REQ-PROV-1 (messages + tools constraint) is not violated. Per `verify-report` this was a ⚠️ design deviation; per the orchestrator's final-state fact it is an approved extension — recorded as resolved-approved, not open.
2. **bash-on-Windows open question RESOLVED**: the `- [ ] bash on Windows dev machine` item in design.md is closed. This host has MSYS2 `C:\msys64\usr\bin\bash.exe`; real-bash scenarios (echo, exit 1, cat, sleep+timeout) passed at runtime. Note: on this Windows host the System32/WindowsApps `bash.exe` (WSL stub) appears earlier in PATH; the tool's `requireBash` detection skips those stubs and locates the real MSYS2 binary (corroborated by runtime test results in verify-report).
3. **2 symlink tests environment-skipped** (no symlink privilege on this Windows host): `read_file`/`write_file` symlink-escape branches (D11) not exercised at runtime here. Follow-up: run on CI/Linux. Per `verify-report` warning #4 + suggestion #2.
4. **`cmd/kui/main.go` 0.0% `go test -cover` is a subprocess-test attribution artifact**, not missing behavior: 4 CLI subprocess tests pass (no-prompt exit 2, missing-key exit 1, provider 500 exit 1, fake provider exit 0).
5. **`.gitignore` fixed in `28c56c8`**: bare `kui` pattern became root-anchored `/kui` (the bare pattern was ignoring `cmd/kui/`).

## Snapshot-Derived Claims (attributed, not terminal facts)

- Coverage figures per `verify-report` at verification time: core 84.4%, tools 86.5%, openai 86.2%, cmd/kui 0.0% (artifact, see above); changed-file average ~83% excluding the cmd/kui artifact.
- Warnings carried from `verify-report` (non-blocking): apply-progress TDD evidence table overwritten by topic upsert (Revisions: 2 — only work unit 3 table survives; mitigated by RED files on disk + GREEN execution); `write_file.go` 69.2% and `errors.go` 20.0% coverage (Error() formatting).
- Suggestions carried: add error-message-string assertions for typed errors; run symlink tests on a symlink-capable host; `go.sum` absent is correct for a zero-dependency module; live end-to-end run needs a real API key.

## Intentional-Warnings Markers

None — archive proceeded under ordinary repository policy. No intentional partial archive, no stale-checkbox reconciliation, no destructive merge (all specs additive, greenfield).

## Gates

- **Task Completion Gate**: passed — 21/21 checked in persisted `tasks.md`, no reconciliation needed.
- **CRITICAL gate**: passed — 0 CRITICAL findings in verify-report.
- **Native Review Receipt Gate**: `reviewGate` structurally absent (no state.yaml, no review artifacts for this candidate) — archive proceeded under ordinary repository policy.
- **Action Context Guard**: no `workspace-planning` mode; operations confined to `openspec/`.

## Mechanical Copy Verification

- 4 spec syncs: `diff -r` exit 0, empty output, each (source delta vs. temp copy before rename).
- Archive move: recursive snapshot before move; `diff -r` snapshot vs. archived folder exit 0, empty output; source directory confirmed gone. `archive-report.md` is additive (written after the readback).
- Archive contents verified present: `proposal.md`, `design.md`, `tasks.md`, `verify-report.md`, `_meta.yaml`, `specs/{agent-loop,agent-tools,provider-openai-compatible,agent-cli}/spec.md`.
- Active `openspec/changes/` no longer contains this change.

## Audit Trail

Archived folder is immutable (audit trail). Source-of-truth main specs under `openspec/specs/` now reflect the shipped behavior. Engram copy of this report: topic key `sdd/agent-foundation/archive-report`, observation ID `18334` (sync `obs-408c53c3ac4194be`), `capture_prompt: false`.
