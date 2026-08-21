# Archive Report — tui-opencode-full-parity

**Archived**: 2026-08-21 | **Store**: openspec | **Actor**: orchestrator (manual archive; sdd-archive dispatch unavailable this session)

## Final State at Close

- **Tasks**: 37/37 complete (`tasks.md`).
- **Verification**: PASS WITH WARNINGS — native validator admitted the typed envelope (`gentle-ai.verify-result/v1`); 28/28 requirements and 63/63 scenarios proven with runtime evidence; full suite 30 packages green, vet/build clean.
- **Correction shipped during verification** (commit `3ca423c`): narrow-mode sidebar overlay was dead code (`_ =` discards) since PR2/PR3; fixed via strict TDD red→green with `applySidebarOverlay()` + backdrop strip, plus `app_overlay_test.go`, `app_golden_test.go`, and goldens `internal/tui/testdata/app_{80,120,160}.txt`. This resolves the two coverage gaps noted at verification time (APP-2 narrow overlay render, APP-10 app layout goldens) — they are CLOSED, not pending.
- **Budget exception accepted**: attempt ordinal 2 settled `passed` with `changed_line_budget_exceeded: true` (472/400 lines incl. correction + artifacts); maintainer reset `req-reset-verify-final-20260821-01` accepted the overage. Per-PR size:exception slices (2972/1411/1791/2353) were maintainer-approved under auto-chain feature-branch-chain.
- **Review gate**: receipt-driven development disabled clone-local before close (user decision); reviewGate structurally absent; delivery under ordinary repository policy.
- **Open follow-ups (non-blocking)**: repo-wide gofmt/EOL normalization (~35 real formatting diffs beyond CRLF noise); diff golden filename wording vs TOOL-4 spec text; optional sidebar toggle binding for narrow mode.

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| tui-app | Updated | MODIFIED REQ-TUI-APP-2/6/7 replaced; ADDED APP-8/9/10 appended (10 reqs / 21 scenarios total) |
| tui-chat | Updated | CHAT-1 style-parity sentence+scenario merged into existing scope; MODIFIED CHAT-2 replaced; ADDED CHAT-4/5/6 appended (6 reqs / 15 scenarios) |
| tui-dialog-overlay | Created | Full mechanical copy (canonical did not exist); 4 reqs / 11 scenarios |
| tui-home | Updated | MODIFIED HOME-1..4 replaced; delta HOME-5 "Header Suppression and Shell Mode" renumbered to canonical REQ-TUI-HOME-7 (HOME-5 Prompt Submission and HOME-6 Keyboard Shortcuts already existed); traceability note recorded in spec (7 reqs / 15 scenarios) |
| tui-theme-system | Created | Full mechanical copy (canonical did not exist); 5 reqs / 10 scenarios |
| tui-tool-view | Updated | MODIFIED TOOL-1 replaced; ADDED TOOL-3/4 appended (4 reqs / 10 scenarios) |

Copy integrity: both full copies and the archive move verified by empty `git diff --no-index` readback (exit 0, no content differences).

## Archive Contents

- proposal.md ✅
- explore.md ✅
- design.md ✅
- specs/ ✅ (6 delta capability specs)
- tasks.md ✅ (37/37)
- apply-progress.md ✅
- verify-report.md ✅ (typed envelope, pass_with_warnings)

## SDD Cycle Complete

Planned → implemented (4 PRs + bounded correction) → verified (28/28 reqs, 63/63 scenarios) → archived. Canonical source of truth now lives under `openspec/specs/{tui-app,tui-chat,tui-dialog-overlay,tui-home,tui-theme-system,tui-tool-view}/spec.md`.
