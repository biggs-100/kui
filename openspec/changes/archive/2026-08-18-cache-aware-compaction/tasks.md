# Tasks: Cache-Aware Compaction

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 340-420 |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Classifier) → PR 2 (Compactor + Builder Integration) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Classifier: ClassifyMessages with system/profile/tool-pair logic | PR 1 | `go test ./internal/core/... -run TestClassify` | N/A — pure function, no runtime harness needed | Revert classifier.go + classifier_test.go |
| 2 | Compactor + Builder integration: protected-aware compaction + loop.go wiring | PR 2 | `go test ./internal/core/... -run TestCompactor` | `go test ./internal/core/... -run TestCacheAware` | Revert compaction.go, loop.go, cache_hints.go changes |

## Phase 1: Classifier (New File)

- [x] 1.1 RED — Create `internal/core/classifier_test.go` with `TestClassifySystemMessagesProtected`: table-driven test asserting `ClassifyMessages([]Message{{Role: RoleSystem, Content: "sys"}})` returns `Protected=true` for system role
- [x] 1.2 GREEN — Create `internal/core/classifier.go` with `ClassifiedMessage` struct and `ClassifyMessages` function implementing system role check
- [x] 1.3 RED — Add `TestClassifyProfileMarkersProtected` to classifier_test.go: messages containing "Profile switched" in content classified as protected
- [x] 1.4 GREEN — Add profile marker detection to `ClassifyMessages`: `strings.Contains(content, "Profile switched")`
- [x] 1.5 RED — Add `TestClassifyToolPairPreservation` to classifier_test.go: assistant message with `ToolCall` + matching `ToolCallID` tool result both classified as protected when in keep window
- [x] 1.6 GREEN — Add tool pair logic: scan forward for matching `ToolCallID` when including assistant with `ToolCall`

## Phase 2: Compactor Integration

- [x] 2.1 RED — Add `TestClassifyCompactorExcludesProtectedFromBudget` to compaction_test.go: protected messages (system, profile) excluded from token budget calculation
- [x] 2.2 GREEN — Modify `Compact()` to call `ClassifyMessages`, subtract protected tokens from `maxTokens` before split-point calculation
- [x] 2.3 RED — Add `TestClassifyCompactorPreservesProtectedPrefix` to compaction_test.go: output ordering is `[protected] → [summary] → [tail]`
- [x] 2.4 GREEN — Assemble output as `protected + summary + tail` in `Compact()`

## Phase 3: Builder Integration

- [x] 3.1 RED — Add `TestClassifyCacheBuilderCalledAfterCompaction` to cache_hints_test.go: `BuildRequest` called with protected messages as prefix
- [x] 3.2 GREEN — Add `BuildRequest(protected, compacted, currentTurn)` method to `CacheAwareRequestBuilder` in cache_hints.go
- [x] 3.3 RED — Add `TestClassifyCacheOptimalOrdering` to cache_hints_test.go: final output order is system → history → current turn

## Phase 4: Edge Cases

- [x] 4.1 RED — Add `TestClassifyAllMessagesProtected` to compaction_test.go: when all messages are system/profile, compaction returns original messages unchanged
- [x] 4.2 RED — Add `TestClassifyEmptyHistory` to compaction_test.go: empty message slice returns empty without error
- [x] 4.3 RED — Add `TestClassifyMultipleSystemMessages` to compaction_test.go: multiple system messages (from profile switches) all appear at prefix in insertion order

## Key Learnings

1. Tool call/result pairs must be preserved atomically to avoid provider errors on orphaned results.
2. Token budget for compactable messages equals total tokens minus protected tokens, not the original maxTokens.
3. ClassifyMessages should be pure and testable in isolation — no provider dependency.
4. Profile marker detection uses content string matching ("Profile switched") rather than role alone.
5. CacheAwareRequestBuilder.BuildRequest takes three separate slices (protected, compacted, currentTurn) for explicit ordering control.
