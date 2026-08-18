# Proposal: Cache-Aware Compaction

## Intent

Compaction destroys OpenAI automatic prefix caching. When `Compactor.Compact` runs, it summarizes old messages into a single `role: "user"` summary at position 0, replacing the stable system-prompt prefix that caching depends on. After compaction, cache hit ratio drops to ~0% — every request re-processes the full system prompt from scratch. The existing `cache_hints.go` code (`SortMessagesForCache`, `CacheAwareRequestBuilder`) is never called in production. This change wires compaction to preserve cache-optimal message ordering.

## Current Gap

1. System prompt (largest cacheable component) gets summarized away
2. Summary inserted as `role: "user"` at position 0 — changes entire byte prefix
3. OpenAI prefix caching relies on byte-identical prefix; compaction makes it different every loop iteration
4. `CacheAwareRequestBuilder` exists but is disconnected from `loop.go` and `requestMessages`

## Scope

### In Scope
- Protect system messages from compaction (never summarized)
- Preserve cache-optimal message order: system → compacted summary → preserved tail
- Integrate `CacheAwareRequestBuilder` into the provider request path
- Protect tool call + result pairs within the keep window from partial summarization
- Add compaction + cache integration tests

### Out of Scope
- Provider-specific cache breakpoint hints (future optimization)
- Semantic deduplication of cached content
- Cost tracking for cache hits vs misses
- Changing compaction token budgets (120K/8K stay as-is)

## Capabilities

### New Capabilities
- `cache-aware-compaction`: Protected message preservation, cache-optimal ordering after compaction, `CacheAwareRequestBuilder` integration into production request path

### Modified Capabilities
- `agent-loop`: Compaction trigger passes system prompt and tool definitions to compactor for protected-message awareness

## Approach

Extend `Compactor` to accept a list of protected messages (system prompt, tool definitions) that are never summarized and always placed at the prefix. After compaction, the output order becomes: `[protected prefix] → [compaction summary] → [preserved tail]`. Wire `CacheAwareRequestBuilder` into `loop.go` so it assembles the final request using this ordering.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/core/compaction.go` | Modified | Protected messages, cache-optimal output ordering |
| `internal/core/cache_hints.go` | Modified | `CacheAwareRequestBuilder` wired into production |
| `internal/core/loop.go` | Modified | Pass system/tools to compactor; use request builder |
| `internal/core/compaction_test.go` | Modified | Protected-message and ordering tests |
| `internal/core/cache_hints_test.go` | Modified | Integration with compaction flow |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Provider-specific cache behavior differs | Med | Design for prefix stability; provider hints are future work |
| Summary quality degrades with protected prefix | Low | Summary prompt unchanged; only split point moves |
| Token budget inaccurate after protection | Low | Protected tokens excluded from compaction budget |

## Rollback Plan

Revert `Compactor.Compact` to current behavior (no protected messages). Remove `CacheAwareRequestBuilder` wiring from `loop.go`. Cache hints code stays as dead code — no production impact.

## Dependencies

Internal: existing `Compactor` interface, `CacheAwareRequestBuilder`, `estimateTokens`.

## Success Criteria

- [ ] System prompt preserved at position 0 after compaction
- [ ] `EstimateCacheSavings` returns >0.5 after compaction (currently ~0)
- [ ] Tool call + result pairs within keep window stay intact
- [ ] All existing compaction tests pass
- [ ] New tests: protected messages, ordering, integration
