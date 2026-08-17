# agent-loop-streaming Specification

## Purpose

Extends the agent loop (REQ-LOOP-1 through REQ-LOOP-7) with a streaming path that processes `StreamChunk` channels from `StreamingProvider` instead of waiting for a complete response.

## MODIFIED Requirements

### Requirement: REQ-LOOP-1 — Loop Execution

The loop MUST accept a user prompt, send it to the provider port, and return the provider's final answer once the provider emits a response without tool calls. The loop MUST operate in two levels: an inner level that alternates provider requests with tool execution and steering-queue draining, and an outer level that drains the follow-up queue only when the inner level would otherwise stop. When the provider implements `StreamingProvider`, the loop MUST use the streaming path: call `StreamChat`, consume chunks from the channel, forward text deltas to the consumer, execute tool calls after stream completion, and then loop. Queued messages MUST be additive: they extend the turn sequence without changing the termination contract.
(Previously: loop used only synchronous `Provider.Chat()` — no streaming path.)

#### Scenario: Direct answer without tools (unchanged)

- GIVEN a provider configured to answer without tool calls
- WHEN the loop runs with the prompt "hello"
- THEN the loop returns the provider's answer content

#### Scenario: Multi-step tool resolution (unchanged)

- GIVEN a provider that first requests a tool call and then answers
- WHEN the loop runs with a prompt requiring that tool
- THEN the loop dispatches the tool, feeds the result back, and returns the final answer

#### Scenario: Streaming direct answer

- GIVEN a `StreamingProvider` returning text chunks then Done
- WHEN the loop runs with the prompt "hello"
- THEN text deltas are forwarded to the consumer during streaming
- AND the final answer is returned after stream completion

#### Scenario: Streaming with tool calls

- GIVEN a `StreamingProvider` returning text chunks, then tool call chunks, then Done
- WHEN the loop runs
- THEN text deltas are forwarded during streaming
- AND tool calls are executed after the stream completes
- AND the loop continues with the tool results

### Requirement: REQ-LOOP-8 — Streaming Provider Detection

The loop MUST detect `StreamingProvider` via type assertion on the provider. If the provider implements `StreamingProvider`, the loop MUST use the streaming path. If not, the loop MUST fall back to the synchronous `Chat()` path.

#### Scenario: Streaming path selected

- GIVEN a provider implementing `StreamingProvider`
- WHEN the loop starts a turn
- THEN `StreamChat` is called instead of `Chat`

#### Scenario: Synchronous fallback

- GIVEN a provider implementing only `Provider`
- WHEN the loop starts a turn
- THEN `Chat` is called (existing behavior unchanged)

### Requirement: REQ-LOOP-9 — Chunk Forwarding

The loop MUST forward `StreamChunk` text deltas to the consumer via the observer's `OnTextDelta` method. The loop MUST NOT block on observer calls — if the observer is nil, deltas are silently consumed.

#### Scenario: Deltas forwarded to observer

- GIVEN a loop with an observer and a streaming provider
- WHEN text chunks arrive on the channel
- THEN `OnTextDelta` is called for each text delta

#### Scenario: Nil observer consumes silently

- GIVEN a loop with a nil observer
- WHEN text chunks arrive on the channel
- THEN deltas are consumed without error and no panic occurs

### Requirement: REQ-LOOP-10 — Post-Stream Tool Execution

Tool calls received during streaming MUST be accumulated and executed only after the stream completes (channel closed). Phase 1 does NOT support streaming tool call execution — all tool calls are batch-processed post-stream.

#### Scenario: Tool calls executed after stream

- GIVEN a stream emitting tool call chunks followed by Done
- WHEN the channel closes
- THEN accumulated tool calls are executed in order

#### Scenario: No tool calls during streaming

- GIVEN a stream emitting only text deltas
- WHEN the channel closes
- THEN no tool execution occurs and the answer is returned directly

### Requirement: REQ-LOOP-11 — Streaming with Queues

Steering and follow-up queues MUST work identically in the streaming path. Queued messages are drained between stream turns, not during an active stream.
(Previously: queues operated only in the synchronous path.)

#### Scenario: Steering message between streaming turns

- GIVEN a queued steering message and a streaming provider
- WHEN a stream completes and a new turn begins
- THEN the steering message is injected before the next `StreamChat` call

#### Scenario: Follow-up queue drains after streaming inner loop

- GIVEN follow-up messages queued during a streaming turn
- WHEN the inner streaming loop would otherwise stop
- THEN follow-up messages are drained and new streaming turns begin
