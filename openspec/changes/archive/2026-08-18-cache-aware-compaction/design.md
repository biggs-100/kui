# Design: Cache-Aware Compaction

## Technical Approach

Extend `Compactor` to classify messages as **protected** (system prompts, profile markers) or **compactable** (conversation history). Protected messages are never summarized and always placed at the message prefix. After compaction, output order becomes: `[protected system messages] → [compaction summary] → [preserved tail]`. Wire `CacheAwareRequestBuilder` into `loop.go` so every provider request uses cache-optimal ordering. Tool call + result pairs within the keep window are preserved atomically.

## Architecture Decisions

### Decision: Protected Message Classification

**Choice**: Add `Protected bool` field to a `ClassifiedMessage` wrapper; classifier function tags messages before compaction.

**Alternatives considered**:
- Filter by role only → Rejected: profile marker messages have `RoleSystem` but contain metadata; role alone is insufficient
- Hardcode in Compact → Rejected: violates single-responsibility; classification logic should be testable in isolation

**Rationale**: A classifier function (`ClassifyMessages`) returns `[]ClassifiedMessage` with a `Protected` flag. This is pure, testable, and extensible. The compactor consumes classified messages without knowing classification rules.

### Decision: Tool Call Pair Preservation

**Choice**: When an assistant message with `ToolCall` is in the keep window, scan forward for the matching `ToolCallID` tool-result message and include both.

**Alternatives considered**:
- Always keep entire tool groups → Rejected: wastes keep budget on old tool results
- Drop orphaned tool results → Rejected: provider rejects tool results without matching calls

**Rationale**: Partial tool call/result pairs cause provider errors. The existing `keepTokens` walk already handles the tail boundary; we adjust it to snap forward when a tool call is included.

### Decision: CacheAwareRequestBuilder Integration Point

**Choice**: Wire into `loop.go` at the point where `messages` are passed to `Provider.Chat` / `StreamingProvider.StreamChat`. Build the cache-aware prefix from: system prompt (extracted from protected messages) + compacted history + current turn.

**Alternatives considered**:
- Replace Compactor entirely → Rejected: Compactor is well-tested; we extend, not replace
- Add as hook → Rejected: hooks fire before compaction; ordering must happen after

**Rationale**: Integration at the request boundary (lines 120-124 of loop.go) ensures all messages are cache-ordered regardless of compaction state.

### Decision: Token Budget Adjustment

**Choice**: Subtract protected message tokens from `maxTokens` before split-point calculation. Protected tokens are "free" in the compaction budget.

**Alternatives considered**:
- Include protected in budget → Rejected: would shrink the compactable window, reducing summary quality
- Ignore budget impact → Rejected: system prompts can be 10K+ tokens; ignoring them risks over-budget requests

**Rationale**: Protected messages are never summarized, so their tokens shouldn't consume the compaction budget. This preserves the effective compactable window size.

## Data Flow

```
Agent.Run (loop.go)
    │
    ├─ Messages arrive (History + user prompt)
    │
    ├─ Compactor.NeedsCompaction(messages)
    │   └─ Counts tokens of ALL messages (including protected)
    │       → Returns true if total > maxTokens
    │
    ├─ Compactor.Compact(ctx, messages)
    │   ├─ ClassifyMessages(messages) → []ClassifiedMessage
    │   │   └─ system role / profile markers → Protected=true
    │   │
    │   ├─ Separate: protected[] + compactable[]
    │   │
    │   ├─ Adjust budget: effectiveMax = maxTokens - protectedTokens
    │   │
    │   ├─ Find split point in compactable[] (existing logic)
    │   │   └─ Snap forward for orphaned tool call/result pairs
    │   │
    │   ├─ Head (compactable[0:split]) → summarize via provider
    │   │
    │   ├─ Tail (compactable[split:]) → keep verbatim
    │   │
    │   └─ Assemble: [protected] + [summary] + [tail]
    │
    ├─ CacheAwareRequestBuilder.BuildCachePrefix()
    │   ├─ System prompt (from protected messages)
    │   ├─ History (compacted messages, excluding system)
    │   └─ Current turn (latest user message)
    │
    └─ Provider.Chat(messages)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/core/compaction.go` | Modify | Add `ClassifiedMessage` type, `ClassifyMessages` function, tool-pair preservation logic, protected-token budget adjustment |
| `internal/core/cache_hints.go` | Modify | Add `BuildRequest` method to `CacheAwareRequestBuilder` that assembles messages in cache-optimal order from classified input |
| `internal/core/loop.go` | Modify | Wire `CacheAwareRequestBuilder` before provider calls; pass system/tools to compactor |
| `internal/core/compaction_test.go` | Modify | Add tests: protected message classification, tool-pair preservation, token budget adjustment |
| `internal/core/cache_hints_test.go` | Modify | Add integration test: compaction + cache ordering |

## Interfaces / Contracts

```go
// ClassifiedMessage wraps a Message with classification metadata.
type ClassifiedMessage struct {
    Message
    Protected bool // true = never summarize, always at prefix
}

// ClassifyMessages tags messages as protected or compactable.
// Rules:
//   - RoleSystem messages are protected
//   - Messages containing "Profile switched" in content are protected
//   - All other messages are compactable
func ClassifyMessages(messages []Message) []ClassifiedMessage

// Compact now accepts classified messages for protected-aware compaction.
func (c *Compactor) Compact(ctx context.Context, messages []Message) ([]Message, error)

// BuildRequest assembles messages in cache-optimal order from classified input.
// Returns: [system prefix] + [compacted history] + [current turn]
func (b *CacheAwareRequestBuilder) BuildRequest(
    protected []Message,
    compacted []Message,
    currentTurn []Message,
) []Message
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `ClassifyMessages`: system → protected, profile marker → protected, user → compactable | Table-driven, assert `Protected` flag |
| Unit | Tool-pair preservation: assistant with ToolCall + matching ToolCallID in keep window | Table-driven, assert both messages in tail |
| Unit | Token budget adjustment: protected tokens excluded from compaction budget | Mock provider, assert split point |
| Unit | `BuildRequest` ordering: system → history → current turn | Assert message positions |
| Integration | Full compaction cycle: messages → classify → compact → BuildRequest → verify ordering | Mock provider, assert final message sequence |
| Integration | Cache savings: `EstimateCacheSavings` > 0.5 after compaction | Use real token estimates |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. The change is internal to the compaction pipeline. Existing sessions that have already been compacted are unaffected — the next compaction will use the new logic. The `CacheAwareRequestBuilder` wiring is opt-in via the existing `Agent.Compactor` field (nil disables).

## Open Questions

- [ ] Should `ClassifyMessages` be exported or package-internal? Exported enables testing from `core_test` package; package-internal keeps the API surface minimal. Recommendation: package-internal (same package tests).
- [ ] How to handle multiple system messages from profile switches? The proposal says "Profile switched" messages are metadata — should they be grouped at prefix or interleaved? Recommendation: all system messages at prefix, preserving insertion order.
