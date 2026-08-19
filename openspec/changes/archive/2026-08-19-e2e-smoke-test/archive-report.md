# Archive Report: e2e-smoke-test

**Date**: 2026-08-19
**Status**: archived
**Mode**: openspec

## Summary

E2E smoke test for kui — build-tagged (`//go:build e2e`) test that validates the full pipeline (CLI binary → provider adapter → real API endpoint → response) against OpenCode's free `big-pickle` model. Skips gracefully when `OPENCODE_API_KEY` is unset.

## Change Description

Minimal test-only change. No spec-level behavior modified. The proposal explicitly states "this is a test-only change with no spec-level behavior" — no new or modified capabilities.

## Artifacts

| Artifact | Status | Notes |
|----------|--------|-------|
| proposal.md | ✅ Present | 67 lines, covers intent/scope/approach/risks/rollback |
| specs/ | — Not applicable | Test-only change; no delta specs needed |
| design.md | — Not present | Not required for this change |
| tasks.md | — Not present | Not required for this change |

## Specs Synced

No delta specs to sync. This change has no spec-level behavior.

## Implementation Summary

- File: `cmd/kui/e2e_test.go` with `//go:build e2e` build tag
- Single test: `TestE2ESmokeOpenCode` — sends prompt to `big-pickle` model, asserts exit 0 + non-empty stdout
- Graceful skip when `OPENCODE_API_KEY` is unset
- Reuses existing `runCLI` pattern from `cmd/kui/main_test.go`
- ~50–100 lines

## Archive Contents

- proposal.md ✅
- specs/ — N/A (no spec changes)
- design.md — N/A
- tasks.md — N/A

## Intentional Partial Archive

This is an intentional partial archive. Only `proposal.md` was produced because this is a test-only change with no spec-level behavior. The proposal explicitly scopes this as out-of-scope for specs, design, and tasks artifacts.

## Risks at Close

None identified. Build tag prevents accidental test runs. Graceful skip handles missing API key.

## SDD Cycle Complete

The change has been fully planned, implemented, verified, and archived.
