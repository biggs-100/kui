# Archive Report: provider-selection

**Archived**: 2026-08-17
**Change**: provider-selection
**Mode**: openspec
**Verdict**: PASS

## Summary

Added a provider registry and selection system to kui. Users can now switch between OpenAI-compatible providers via `--provider` / `-p` CLI flag, `provider:` field in profile.yaml, or `KUI_PROVIDER` environment variable. Includes fail-fast API key validation and graceful thinking degradation.

## Artifacts

| Artifact | Path | Status |
|----------|------|--------|
| proposal.md | `openspec/changes/archive/2026-08-17-provider-selection/proposal.md` | ✅ |
| design.md | `openspec/changes/archive/2026-08-17-provider-selection/design.md` | ✅ |
| tasks.md | `openspec/changes/archive/2026-08-17-provider-selection/tasks.md` | ✅ (38/38 complete) |
| verify-report.md | `openspec/changes/archive/2026-08-17-provider-selection/verify-report.md` | ✅ (PASS) |
| specs/ | 5 domain delta specs | ✅ |

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| provider-selection | Created | Full spec (4 requirements, 8 scenarios) — REQ-SEL-1 through REQ-SEL-4 |
| cli-flags | Updated | MODIFIED REQ-CLI-10: added `Provider string` to Options struct, 2 new scenarios |
| profile-runtime | Updated | MODIFIED REQ-PROFILE-1: added `provider` field to profile declaration, 2 new scenarios |
| provider-openai-compatible | Updated | MODIFIED REQ-PROV-3: per-provider base URL env var, 1 new scenario |
| thinking-provider | Updated | ADDED REQ-THINK-13: thinking capability check, 2 new scenarios |

## Verification

- **Tests**: 17 packages passed, 0 failed, 0 skipped
- **Build**: clean
- **Vet**: clean
- **Scenarios**: 19/19 compliant
- **CRITICAL issues**: 0
- **WARNING issues**: 0

## Functions Extracted

- `ResolveProvider()` — layered resolution chain (flag → profile → env → default)
- `CreateProvider()` — registry lookup + fail-fast key validation
- `WarnThinkingDegradation()` — thinking capability check with io.Writer injection

## Mechanical Copy Verification

```
All 10 files IDENTICAL (MD5 match) — empty diff
```

Byte-identity verified between pre-move snapshot and archived folder. No truncation or alteration detected.
