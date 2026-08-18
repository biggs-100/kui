# Cache Builder Integration Specification

## Purpose

Wires `CacheAwareRequestBuilder` into the production agent loop so every provider request uses cache-optimal message ordering. The builder assembles the final request from protected messages, compacted history, and the current turn.

## Requirements

### REQ-CBI-01: Builder Called After Compaction
**Priority**: P0
**Description**: `CacheAwareRequestBuilder.BuildRequest` SHALL be called in `loop.go` after compaction completes and before the provider is invoked. The builder receives the compaction output and assembles the final request messages.
**Acceptance Criteria**: Provider calls always go through `BuildRequest` when compaction has occurred.
**Scenarios**:

- GIVEN a loop with compaction triggered
- WHEN compaction completes
- THEN `BuildRequest` is called with the compaction output
- AND the provider receives the builder's output, not the raw compaction output

- GIVEN a loop where compaction did NOT trigger
- WHEN the loop sends a provider request
- THEN `BuildRequest` is called with the full history (no compaction)

### REQ-CBI-02: Builder Input Contract
**Priority**: P0
**Description**: `BuildRequest` SHALL receive three separate slices: protected messages, compacted history (excluding system), and the current turn (latest user message).
**Acceptance Criteria**: `BuildRequest(protected, compacted, currentTurn)` assembles them in cache-optimal order.
**Scenarios**:

- GIVEN protected messages, compacted history, and a current turn
- WHEN `BuildRequest` is called
- THEN the output starts with protected messages
- AND compacted history follows
- AND the current turn is last

- GIVEN empty compacted history
- WHEN `BuildRequest` is called with protected messages and current turn
- THEN the output is protected messages followed by current turn

### REQ-CBI-03: Cache-Optimal Final Ordering
**Priority**: P0
**Description**: The final message ordering delivered to the provider SHALL be cache-optimal: system messages → assistant messages → tool messages → user messages. This ordering maximizes OpenAI prefix cache hit rates.
**Acceptance Criteria**: Provider receives messages in the order: system → assistant → tool → user.
**Scenarios**:

- GIVEN a mixed history with system, assistant, tool, and user messages
- WHEN `BuildRequest` assembles the final output
- THEN system messages appear first
- AND assistant messages follow system messages
- AND tool messages follow assistant messages
- AND user messages appear last

- GIVEN a compacted history with summary at position 0
- WHEN `BuildRequest` assembles the final output
- THEN the summary is placed in the correct cache-optimal position

## Out of Scope

- Provider-specific cache breakpoint hints (future optimization)
- Cache hit/miss metrics or cost tracking
- Semantic deduplication of cached content
- Modifying the compaction pipeline itself
