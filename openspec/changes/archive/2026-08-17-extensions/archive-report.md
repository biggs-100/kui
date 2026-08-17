# Archive Report: extensions

## Summary

| Field | Value |
|-------|-------|
| Change | extensions |
| Archive Date | 2026-08-17 |
| Branch | feat/extensions/3-discovery |
| PRs | #29–#31 (pending review) |
| Tasks | 28/28 complete |
| Verify | PASS WITH WARNINGS (20/20 req, 44/44 scenarios) |
| Build | go build ./cmd/kui — exit 0 |
| Tests | 13 packages, 0 failures, race detector clean |

## Verification Details

Per `verify-report.md` (at verification time):

- **Compliance**: 43/44 scenarios fully compliant, 1/44 partial (REQ-EXT-2 duplicate-name scenario — mock is no-op; real detection delegated to concrete tools.Registry)
- **Warnings**:
  1. Missing TDD apply-progress artifact — all 28 tasks have tests but formal TDD chain-of-evidence absent
  2. REQ-EXT-2 duplicate-name test is partial (mock returns nil)
- **Suggestion**: HookRegistry.Emit uses `fmt.Errorf` instead of `HookError` type — consider for consistency

No CRITICAL issues. No blockers.

## Design Decisions

| # | Decision | Choice |
|---|----------|--------|
| D1 | Extension interface | `Name()`, `Init(ExtensionAPI) error`, `Shutdown() error` |
| D2 | ExtensionAPI scope | `RegisterTool`, `RegisterHook`, `RegisterCommand` (stubbed) |
| D3 | HookRegistry structure | `map[string][]HookHandler` with registration-order slice |
| D4 | HookContext mutability | Mutable struct with `Messages()`/`SetMessages()`, `Block()`/`IsBlocked()` |
| D5 | Hook injection points | 3 hooks (not 8): before_provider_request, before_tool_execution, after_tool_execution |
| D6 | Discovery mechanism | Compiled-in via `init()` + `extensions.Register(ext)` |
| D7 | Backward compatibility | Nil-safe `*HookRegistry` on `Agent` struct |
| D8 | Panic recovery | Per-handler `recover()` wrapping (same as Observer pattern) |

## Key Files Created

| File | Description |
|------|-------------|
| `internal/core/extension.go` | Extension interface, ExtensionAPI, HookHandler, HookContext |
| `internal/core/hook_registry.go` | HookRegistry struct: Register, Emit, HasHooks |
| `internal/core/hook_context.go` | Concrete hookContext implementing HookContext |
| `internal/core/errors.go` | HookError type (event name + wrapped error) |
| `internal/adapters/extensions/registry.go` | Register, LoadAll, ShutdownAll functions |
| `internal/extensions/example/example.go` | Example extension implementing Extension interface |

## Key Files Modified

| File | Changes |
|------|---------|
| `internal/core/loop.go` | Added `Hooks *HookRegistry` field; emit hooks at 3 lifecycle points |
| `cmd/kui/main.go` | Blank import for example extension package |

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| extension-system | Created | 6 requirements, 13 scenarios |
| hook-registry | Created | 5 requirements, 10 scenarios |
| agent-loop-hooks | Merged into agent-loop | MODIFIED REQ-LOOP-7, ADDED REQ-LOOP-12/13/14/15 (5 req, 12 scenarios) |
| extension-discovery | Created | 4 requirements, 9 scenarios |

## PR Status

3 stacked PRs on `feat/extensions/3-discovery`:

- **PR #1**: Core ports — Extension, HookRegistry, HookContext interfaces + impl (Slice A)
- **PR #2**: Loop integration — emit hooks at 3 lifecycle points (Slice B)
- **PR #3**: Discovery + example extension (Slice C)

All pending review at archive time.
