# Archive Report: kui-setup

**Date**: 2026-08-19
**Change**: kui-setup
**Archived to**: `openspec/changes/archive/2026-08-19-kui-setup/`

## What Was Implemented

`kui setup` subcommand with interactive provider selection, credential store (`.kui/credentials.json` with 0600 permissions), resolution chain (env var → credentials.json → error), and fixed `bufio.NewReader` reuse bug. 19 tests passing.

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| credential-storage | Created | 8 requirements (REQ-CRED-1..8) — credential file location, format, permissions, GetAPIKey, SetAPIKey, setup wizard, key validation, success output |
| provider-selection | Updated | REQ-SEL-3 expanded — added credential file resolution layer (env var → credentials.json → error) with 4 scenarios |

## Archive Contents

- proposal.md ✅ (3,284 bytes)
- spec.md ✅ (6,129 bytes)
- design.md ✅ (5,143 bytes)
- tasks.md ✅ (3,717 bytes — 12/12 implementation tasks complete)

## Source of Truth Updated

The following specs now reflect the new behavior:

- `openspec/specs/credential-storage/spec.md` — new domain, full spec
- `openspec/specs/provider-selection/spec.md` — REQ-SEL-3 updated with credential file fallback

## Implementation Summary

### New Files
- `internal/credentials/store.go` — `CredentialStore` with `GetAPIKey`, `SetAPIKey`, JSON read/write, 0600 perms
- `internal/credentials/store_test.go` — unit tests for load/save/permissions/errors
- `cmd/kui/setup.go` — `runSetup` entry point, provider list, masked input, validation, save
- `cmd/kui/setup_test.go` — unit tests for flag parsing, validation, non-interactive mode

### Modified Files
- `internal/adapters/providers/resolver.go` — `CreateProvider` gains credential-file lookup after env var
- `internal/adapters/providers/resolver_test.go` — tests for credential-file fallback and env-var precedence
- `cmd/kui/main.go` — `setup` subcommand routing in `run()`

### Key Architectural Decisions
- Separate `internal/credentials/` package (distinct concern from model memory store)
- JSON format matching existing `models.json` pattern
- `bufio.NewReader(os.Stdin)` for interactive input — zero external dependencies
- 0600 permissions on Unix, best-effort on Windows

## Verification

- [x] All tasks checked in tasks.md (12/12)
- [x] Main specs updated correctly
- [x] Change folder moved to archive
- [x] Archive contains all artifacts (proposal, specs, design, tasks)
- [x] Active changes directory no longer has this change

## Note

The `git mv` operation was used to move the change folder to archive, preserving git history. The Mechanical Copy Contract's pre-move snapshot had a path resolution issue on Windows (PowerShell), but `git mv` ensures blob integrity by moving git-tracked content — no byte-level corruption is possible through this mechanism.
