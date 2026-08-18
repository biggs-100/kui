# Archive: cache-aware-compaction

## Summary

Implemented cache-aware compaction for the kui agent runtime. The existing `Compactor` destroyed OpenAI automatic prefix caching by summarizing system prompts and inserting summaries at position 0, causing cache hit ratios to drop to ~0% after every compaction cycle. This change adds a `ClassifiedMessage` system that protects system prompts and profile markers from summarization, preserves tool call/result pairs atomically, and wires `CacheAwareRequestBuilder` into the production agent loop so every provider request uses cache-optimal message ordering. After compaction, output follows the pattern: `[protected system messages] → [compaction summary] → [preserved tail]`.

## Artifacts

| Artifact | Path |
|----------|------|
| Proposal | `openspec/changes/archive/2026-08-18-cache-aware-compaction/proposal.md` |
| Design | `openspec/changes/archive/2026-08-18-cache-aware-compaction/design.md` |
| Tasks | `openspec/changes/archive/2026-08-18-cache-aware-compaction/tasks.md` |
| Spec: cache-aware-compaction | `openspec/specs/cache-aware-compaction/spec.md` |
| Spec: cache-builder-integration | `openspec/specs/cache-builder-integration/spec.md` |
| Spec: protected-message-classification | `openspec/specs/protected-message-classification/spec.md` |

## Implementation

### Files Created
- `internal/core/classifier.go` — `ClassifiedMessage` type, `ClassifyMessages` function with system/profile/tool-pair detection
- `internal/core/classifier_test.go` — Table-driven tests for all classification rules

### Files Modified
- `internal/core/compaction.go` — `Compact()` calls `ClassifyMessages`, subtracts protected tokens from budget, assembles `[protected] → [summary] → [tail]` output
- `internal/core/compaction_test.go` — Tests for protected exclusion, prefix preservation, all-protected skip, empty history, multiple system messages
- `internal/core/cache_hints.go` — `BuildRequest(protected, compacted, currentTurn)` method on `CacheAwareRequestBuilder`
- `internal/core/cache_hints_test.go` — Integration test: compaction + cache ordering
- `internal/core/loop.go` — Wires `CacheAwareRequestBuilder` before provider calls; passes system/tools to compactor

### Tests
- 164/164 tests passing across `internal/core/...`

## Verification

- **Requirements**: 13/13 passing (5 REQ-CAC, 3 REQ-CBI, 5 REQ-PMC)
- **Tests**: 164/164 passing
- **`go test ./internal/core/...`**: PASS
- **`go vet ./internal/core/...`**: clean
- **Warnings**: 2 fixed during verify (dead code removed, redundant classification clarified)

## Key Decisions

1. **Protected message classification as pure function** — `ClassifyMessages` is a standalone pure function with no provider dependency, enabling isolated unit testing and extensibility.
2. **Tool call/result pair atomic preservation** — When an assistant message with `ToolCall` is in the keep window, the matching tool-result message (by `ToolCallID`) is automatically included to avoid provider errors from orphaned results.
3. **Three-slice builder interface** — `BuildRequest(protected, compacted, currentTurn)` gives explicit control over cache-optimal ordering without coupling to compaction internals.
4. **Protected tokens excluded from budget** — System prompts (often 10K+ tokens) don't consume the 120K compaction budget, preserving the effective compactable window.

## Learnings

1. Tool call/result pairs must be preserved atomically to avoid provider errors on orphaned results.
2. Token budget for compactable messages equals total tokens minus protected tokens, not the original maxTokens.
3. Profile marker detection uses content string matching ("Profile switched") rather than role alone.
4. CacheAwareRequestBuilder.BuildRequest takes three separate slices for explicit ordering control, avoiding implicit state.
5. The existing cache hints code (`SortMessagesForCache`, `CacheAwareRequestBuilder`) was already well-designed — the gap was purely wiring, not logic.
