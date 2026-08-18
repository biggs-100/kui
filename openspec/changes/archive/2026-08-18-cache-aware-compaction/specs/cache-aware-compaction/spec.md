# Cache-Aware Compaction Specification

## Purpose

Extends the existing `Compactor` to produce cache-optimal output by excluding protected messages from the compaction token budget and preserving them as a stable prefix. This restores OpenAI automatic prefix caching after compaction.

## Requirements

### REQ-CAC-01: Protected Token Exclusion
**Priority**: P0
**Description**: The compactor SHALL exclude protected message tokens from the compaction token budget. Protected tokens are "free" — they do not consume the 120K compaction budget.
**Acceptance Criteria**: Split-point calculation uses `effectiveMax = maxTokens - protectedTokens`. Protected messages are never summarized.
**Scenarios**:

- GIVEN 150K total tokens with 20K protected
- WHEN `Compact` calculates the split point
- THEN the effective budget is 130K (150K - 20K)
- AND the compactable window spans from the first compactable message to the split point

- GIVEN messages where all tokens are protected (0K compactable)
- WHEN `Compact` is called
- THEN compaction does NOT run (see REQ-CAC-05)

### REQ-CAC-02: Protected Prefix Preservation
**Priority**: P0
**Description**: The compactor output SHALL preserve protected messages as a prefix. Protected messages appear first in the output in their original insertion order, followed by the compaction summary and preserved tail.
**Acceptance Criteria**: Output starts with all protected messages in order, then summary, then compactable tail.
**Scenarios**:

- GIVEN 3 protected messages and 10 compactable messages
- WHEN `Compact` produces output
- THEN the first 3 messages are the protected messages in original order
- AND messages 4+ are the summary and preserved tail

- GIVEN multiple system messages from profile switches
- WHEN `Compact` produces output
- THEN all system messages appear at prefix in insertion order

### REQ-CAC-03: Output Ordering Contract
**Priority**: P0
**Description**: The compactor output ordering SHALL be: `[protected messages] → [compaction summary] → [preserved tail]`. This ordering is mandatory for cache stability.
**Acceptance Criteria**: Final output position 0 is always the first protected message (or summary if no protected messages).
**Scenarios**:

- GIVEN protected messages followed by compactable messages
- WHEN `Compact` returns output
- THEN position 0 is the first protected message
- AND the summary immediately follows all protected messages
- AND the preserved tail follows the summary

- GIVEN no protected messages
- WHEN `Compact` returns output
- THEN position 0 is the summary message
- AND the preserved tail follows the summary

### REQ-CAC-04: Summary Message Format
**Priority**: P1
**Description**: The summary message SHALL have `role: "user"` and its content SHALL begin with the prefix `[session-compaction]`.
**Acceptance Criteria**: Summary message matches `{role: "user", content: "[session-compaction] ..."}`.
**Scenarios**:

- GIVEN a compaction that produces a summary
- WHEN the summary message is created
- THEN its role is "user"
- AND its content starts with "[session-compaction]"

### REQ-CAC-05: All-Protected Skip
**Priority**: P0
**Description**: When ALL messages are classified as protected, compaction SHALL NOT run. The original messages are returned unchanged.
**Acceptance Criteria**: If every message is protected, `Compact` returns the original slice without calling the provider.
**Scenarios**:

- GIVEN messages where all are system-role
- WHEN `Compact` is called
- THEN the original messages are returned unchanged
- AND no provider summarization request is made

- GIVEN messages where all are profile markers
- WHEN `Compact` is called
- THEN the original messages are returned unchanged

## Out of Scope

- Changing compaction token budgets (120K/8K stay as-is)
- Provider-specific cache breakpoint hints
- Summary quality tuning
- Cost tracking for cache hits vs misses
