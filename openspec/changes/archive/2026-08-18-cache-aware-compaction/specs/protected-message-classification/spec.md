# Protected Message Classification Specification

## Purpose

Defines how messages are classified as protected (never summarized) or compactable before entering the compaction pipeline. Classification is a pure function with no provider dependency, enabling isolated unit testing.

## Requirements

### REQ-PMC-01: System Message Protection
**Priority**: P0
**Description**: All messages with `role == "system"` SHALL be classified as protected.
**Acceptance Criteria**: `ClassifyMessages` returns `Protected=true` for every system-role message regardless of content.
**Scenarios**:

- GIVEN a single system message with role "system"
- WHEN `ClassifyMessages` is called
- THEN the result contains one entry with `Protected=true`
- AND the original message content is unchanged

- GIVEN a mix of system and user messages
- WHEN `ClassifyMessages` is called
- THEN only system messages have `Protected=true`
- AND user messages have `Protected=false`

### REQ-PMC-02: Profile Marker Protection
**Priority**: P0
**Description**: Messages containing the substring "Profile switched" in their content SHALL be classified as protected, regardless of their role field.
**Acceptance Criteria**: Any message (system, user, or assistant) with "Profile switched" in content is marked protected.
**Scenarios**:

- GIVEN a system message containing "Profile switched to coder"
- WHEN `ClassifyMessages` is called
- THEN that message is classified as protected

- GIVEN a user message containing "Profile switched"
- WHEN `ClassifyMessages` is called
- THEN that message is classified as protected
- AND a user message without "Profile switched" remains compactable

### REQ-PMC-03: Tool Call Pair Preservation
**Priority**: P0
**Description**: Tool results whose corresponding `tool_call` (matched by `ToolCallID`) is protected SHALL also be classified as protected. When an assistant message with a `ToolCall` is protected, the matching tool-result message MUST be protected too.
**Acceptance Criteria**: For every protected assistant message containing a `ToolCall`, the tool-result message with matching `ToolCallID` is also marked protected.
**Scenarios**:

- GIVEN an assistant message with `ToolCall` (protected) followed by a tool result with matching `ToolCallID`
- WHEN `ClassifyMessages` is called
- THEN both the assistant message and the tool result are classified as protected

- GIVEN an assistant message without `ToolCall`
- WHEN `ClassifyMessages` is called
- THEN classification follows standard rules (role/content based)

### REQ-PMC-04: Default Compactable
**Priority**: P1
**Description**: All messages that are NOT system-role, NOT profile markers, and NOT matched tool results SHALL be classified as compactable (`Protected=false`).
**Acceptance Criteria**: User messages, assistant messages without tool calls, and unmatched tool results are compactable.
**Scenarios**:

- GIVEN a user message
- WHEN `ClassifyMessages` is called
- THEN the message is classified as compactable

- GIVEN an assistant message without ToolCall
- WHEN `ClassifyMessages` is called
- THEN the message is classified as compactable

### REQ-PMC-05: Empty Input Handling
**Priority**: P1
**Description**: An empty input slice SHALL produce empty protected and compactable output slices with no error.
**Acceptance Criteria**: `ClassifyMessages([]Message{})` returns an empty slice without panicking.
**Scenarios**:

- GIVEN an empty message slice
- WHEN `ClassifyMessages` is called
- THEN the result is an empty slice
- AND no error is returned

## Out of Scope

- Semantic deduplication of protected content
- Provider-specific cache breakpoint hints
- Dynamic classification rules beyond role/content matching
