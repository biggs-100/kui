# Proposal: E2E Smoke Test

## Intent

kui has strong CLI unit/integration tests (`runCLI` pattern, `httptest` providers) but zero live-provider E2E tests. We need a smoke test that proves the full pipeline — CLI binary → provider adapter → real API endpoint → response — actually works against a live endpoint. This catches wiring bugs that mock-based tests miss.

## Scope

### In Scope
- New file: `cmd/kui/e2e_test.go` with `//go:build e2e` build tag
- Single smoke test: send a prompt to OpenCode's free `big-pickle` model, assert exit 0 and non-empty stdout
- Skip automatically if `OPENCODE_API_KEY` is unset (skip, not fail)
- Reuse existing `runCLI` pattern from `cmd/kui/main_test.go`

### Out of Scope
- Full E2E test suite (multiple providers, multi-turn, tool use)
- Rate limit handling or retry logic
- Fallback chain testing (provider → fallback)
- CI integration (manual `go test -tags=e2e` only for now)
- Testing non-OpenCode providers

## Capabilities

### New Capabilities
None — this is a test-only change with no spec-level behavior.

### Modified Capabilities
None — no existing requirements change.

## Approach

1. Add `cmd/kui/e2e_test.go` gated behind `//go:build e2e`
2. In `TestE2ESmokeOpenCode`: check `OPENCODE_API_KEY` env, skip if missing
3. Call `runCLI(t, env, "--provider", "opencode", "--model", "big-pickle", "Say hello in 5 words or fewer")`
4. Assert exit code 0, stdout contains a response, no stderr errors
5. ~50–100 lines total

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/kui/e2e_test.go` | New | Build-tagged E2E smoke test |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Network unavailable during test | Medium | `t.Skip` if connection fails; build tag prevents accidental runs |
| API key not set | High | Skip (not fail) when `OPENCODE_API_KEY` is empty |
| Rate limits on free model | Low | Single request per test run; `big-pickle` has generous limits |
| Model/response format changes | Low | Assert presence, not exact content |

## Rollback Plan

Delete `cmd/kui/e2e_test.go`. No other files are modified.

## Dependencies

- OpenCode account with `OPENCODE_API_KEY` set
- `big-pickle` model available on OpenCode provider

## Success Criteria

- [ ] `go test -tags=e2e -run TestE2ESmokeOpenCode ./cmd/kui/` passes with real OpenCode endpoint
- [ ] Test skips gracefully when `OPENCODE_API_KEY` is unset
- [ ] Existing `go test ./...` (no build tag) continues to pass — E2E tests never run in normal CI
- [ ] Test is under 100 lines
