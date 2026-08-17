# tui-streaming Specification

## Purpose

Extends the TUI controller and chat model to detect streaming providers and render text deltas incrementally as they arrive.

## MODIFIED Requirements

### Requirement: REQ-TUI-APP-3 — Concurrency Boundary

`agent.Run` MUST execute on a goroutine separate from the TUI. The loop goroutine MUST NOT mutate UI state directly; all UI updates MUST be dispatched via `tea.Cmd`. No UI work MAY run on the loop's goroutine. When streaming, chunk events MUST be dispatched as individual `tea.Cmd` messages — the loop goroutine MUST NOT write to the Bubble Tea program directly.
(Previously: concurrency boundary existed but did not cover streaming chunk dispatch.)

#### Scenario: Turn events reach the UI (unchanged)

- GIVEN a multi-step turn running on the loop goroutine
- WHEN events are produced
- THEN they are delivered to the UI through `tea.Cmd`

#### Scenario: Streaming chunks dispatched as tea.Cmd

- GIVEN a streaming provider emitting text deltas on the loop goroutine
- WHEN each delta arrives
- THEN it is wrapped in a `tea.Cmd` and sent to the Bubble Tea program
- AND no direct UI mutation occurs from the loop goroutine

### Requirement: REQ-TUI-CHAT-2 — Streaming Answer Rendering

While the provider streams, the chat view MUST render answer content incrementally as events arrive. If the stream fails mid-answer, the view MUST render an error state for that answer and MUST NOT crash the app. The chat model MUST support `AppendChunk(delta string)` for incremental text growth.
(Previously: streaming rendering described but no controller-level detection or dispatch.)

#### Scenario: Incremental render (unchanged)

- GIVEN a provider that streams an answer in chunks
- WHEN chunks arrive
- THEN the answer text grows in the view as each chunk renders

#### Scenario: Stream error mid-answer (unchanged)

- GIVEN a provider whose stream fails partway through an answer
- WHEN the failure occurs
- THEN the partial answer shows an error state
- AND the app keeps running

## ADDED Requirements

### Requirement: REQ-TUI-STREAM-1 — StreamingProvider Detection

The controller MUST detect whether the runner implements `StreamingProvider` via type assertion (`if sp, ok := runner.(StreamingProvider); ok`). If true, the controller MUST use the streaming execution path. If false, the controller MUST fall back to the synchronous path.

#### Scenario: Streaming path activated

- GIVEN a runner implementing `StreamingProvider`
- WHEN the user submits a prompt
- THEN the controller calls `StreamChat` and processes chunks

#### Scenario: Synchronous fallback

- GIVEN a runner implementing only `Provider`
- WHEN the user submits a prompt
- THEN the controller calls `Chat` (existing behavior)

### Requirement: REQ-TUI-STREAM-2 — streamChunkMsg Dispatch

For each `StreamChunk` with `TextDelta` set, the controller MUST emit a `streamChunkMsg{delta}` to the Bubble Tea program. The message MUST be dispatched from the streaming goroutine via `tea.Cmd` — never directly.

#### Scenario: Per-delta message dispatch

- GIVEN a streaming provider emitting 5 text deltas
- WHEN each delta arrives on the channel
- THEN a `streamChunkMsg` is emitted for each delta
- AND the chat model appends the delta to the current answer

#### Scenario: Non-text chunks ignored by TUI

- GIVEN a streaming provider emitting tool call chunks
- WHEN tool call chunks arrive
- THEN no `streamChunkMsg` is emitted (tool calls processed post-stream by loop)

### Requirement: REQ-TUI-STREAM-3 — ChatModel.AppendChunk

The chat model's `AppendChunk(delta string)` method MUST append the delta text to the current streaming answer. This method already exists and MUST be called for each `streamChunkMsg`.

#### Scenario: AppendChunk grows answer text

- GIVEN a chat model with a partial answer "Hello"
- WHEN `AppendChunk(" world")` is called
- THEN the answer text becomes "Hello world"

#### Scenario: AppendChunk on empty answer

- GIVEN a chat model with no current answer
- WHEN `AppendChunk("first")` is called
- THEN the answer text becomes "first"

### Requirement: REQ-TUI-STREAM-4 — Stream Completion

When the stream channel closes, the controller MUST emit a `streamDoneMsg{err}` where `err` is nil on success or the error from the stream. The chat model MUST finalize the current answer on receiving this message.

#### Scenario: Successful stream completion

- GIVEN a stream completing with Done=true
- WHEN the channel closes
- THEN `streamDoneMsg{nil}` is emitted
- AND the answer is finalized in the chat model

#### Scenario: Stream error completion

- GIVEN a stream failing with an error
- WHEN the error chunk is received
- THEN `streamDoneMsg{err}` is emitted
- AND the answer shows an error state
