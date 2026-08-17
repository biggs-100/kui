# provider-streaming Specification

## Purpose

Defines the `StreamingProvider` interface and `StreamChunk` type for real-time token-by-token LLM responses. This is an opt-in capability that extends the existing `Provider` interface without breaking it.

## ADDED Requirements

### Requirement: REQ-STREAM-1 — StreamingProvider Interface

The system MUST define a `StreamingProvider` interface extending `Provider` with a `StreamChat(ctx context.Context, messages []Message, tools []ToolDef) (<-chan StreamChunk, error)` method. The interface MUST be optional — non-streaming callers continue using `Provider.Chat()` unchanged.

#### Scenario: StreamingProvider exposes StreamChat

- GIVEN a provider implementing `StreamingProvider`
- WHEN the caller invokes `StreamChat` with messages and tools
- THEN a read-only channel of `StreamChunk` and no error are returned

#### Scenario: Non-streaming provider does not implement StreamingProvider

- GIVEN a provider implementing only `Provider` (no `StreamChat`)
- WHEN a caller checks `_, ok := provider.(StreamingProvider)`
- THEN the type assertion returns `false`

### Requirement: REQ-STREAM-2 — StreamChunk Type

The system MUST define a `StreamChunk` struct with fields: `TextDelta string`, `ToolCallStart *ToolCallStart`, `ToolCallDelta *ToolCallDelta`, `ToolCallEnd *ToolCallEnd`, `Error error`, `Done bool`, and `Usage *Usage`. Exactly one payload field MUST be set per chunk (mutual exclusivity by convention).

#### Scenario: Text delta chunk

- GIVEN a streaming response containing text tokens
- WHEN the provider emits a text chunk
- THEN the `StreamChunk` has `TextDelta` set and all other payload fields nil/zero

#### Scenario: Done chunk

- GIVEN a completed stream
- WHEN the provider emits the final chunk
- THEN the `StreamChunk` has `Done: true` and all payload fields nil/zero

### Requirement: REQ-STREAM-3 — Channel Lifecycle

The stream channel MUST be closed by the provider after the final chunk (success or error). Consumers MUST NOT send on the channel. The channel MUST have a bounded buffer (default 32).

#### Scenario: Channel closed on success

- GIVEN a streaming provider returning a complete response
- WHEN the final `Done: true` chunk is sent
- THEN the channel is closed and no more chunks arrive

#### Scenario: Channel closed on error

- GIVEN a streaming provider encountering an error mid-stream
- WHEN the error chunk is sent
- THEN the channel is closed immediately after

### Requirement: REQ-STREAM-4 — Context Cancellation

When the provided context is cancelled, the provider MUST stop producing chunks and close the channel. The provider MUST NOT leak goroutines.

#### Scenario: Context cancelled mid-stream

- GIVEN a stream in progress
- WHEN the caller cancels the context
- THEN the channel closes within a bounded time
- AND no goroutine leak occurs

#### Scenario: Context already cancelled before StreamChat

- GIVEN a pre-cancelled context
- WHEN `StreamChat` is invoked
- THEN the method returns an error immediately without opening a stream

### Requirement: REQ-STREAM-5 — Error Propagation

A mid-stream error MUST be delivered as `StreamChunk{Error: err}` followed by channel close. The consumer MUST receive exactly one error chunk maximum per stream.

#### Scenario: Network failure mid-stream

- GIVEN a stream in progress
- WHEN a network error occurs
- THEN the consumer receives `StreamChunk{Error: err}`
- AND the channel is closed

#### Scenario: No duplicate error chunks

- GIVEN a stream error
- WHEN the error chunk is delivered
- THEN no additional chunks follow before channel close
