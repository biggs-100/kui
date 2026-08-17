# observer-streaming Specification

## Purpose

Extends the Observer port (REQ-LOOP-7) with a `OnTextDelta` method for real-time text chunk events during streaming.

## MODIFIED Requirements

### Requirement: REQ-LOOP-7 — Observer Port (nil-safe, stdlib)

The loop MUST expose an optional observer port through which tool-call, tool-result, turn, and text-delta events are published. The port MUST use stdlib types only. When the observer is nil, the loop MUST behave identically to today — all existing loop tests MUST pass unmodified. Observer events MUST NOT influence loop control flow: termination, iteration budget, and event ordering MUST be unchanged regardless of observer behavior or absence.
(Previously: Observer published tool-call, tool-result, and turn events only — no text-delta events.)

#### Scenario: Nil observer is a no-op (unchanged)

- GIVEN a loop constructed with a nil observer
- WHEN the loop runs any existing scenario
- THEN behavior is identical to a loop without an observer

#### Scenario: Tool events published (unchanged)

- GIVEN a loop with an observer attached
- WHEN a tool call and its result occur
- THEN the observer receives the call and result events

#### Scenario: Turn events published (unchanged)

- GIVEN a loop with an observer attached
- WHEN a turn starts and completes
- THEN the observer receives start and completion events

#### Scenario: Observer failure is contained (unchanged)

- GIVEN an observer that panics on an event
- WHEN the loop emits an event
- THEN the panic is contained and the loop continues

## ADDED Requirements

### Requirement: REQ-OBS-STREAM-1 — OnTextDelta Method

The Observer interface MUST include `OnTextDelta(delta string)` for real-time text chunk events. This method MUST be called for each text delta received during streaming. The method MUST be safe to call from any goroutine.

#### Scenario: OnTextDelta called per chunk

- GIVEN an observer attached to a loop with a streaming provider
- WHEN three text deltas arrive during a stream
- THEN `OnTextDelta` is called three times with the correct delta strings

#### Scenario: OnTextDelta with empty delta

- GIVEN a stream producing an empty text delta
- WHEN the chunk arrives
- THEN `OnTextDelta("")` is called (not skipped)

### Requirement: REQ-OBS-STREAM-2 — Nil-Safe OnTextDelta

When the observer is nil, calls to `OnTextDelta` MUST be no-ops. The loop MUST NOT check for nil before every call — a nil-safe wrapper or default implementation MUST handle this.

#### Scenario: Nil observer OnTextDelta is no-op

- GIVEN a loop with a nil observer
- WHEN text deltas arrive during streaming
- THEN no panic occurs and deltas are silently consumed

#### Scenario: Non-nil observer receives all deltas

- GIVEN a loop with a non-nil observer
- WHEN text deltas arrive during streaming
- THEN every delta is delivered to the observer
- AND no deltas are lost

### Requirement: REQ-OBS-STREAM-3 — Observer Failure Containment for OnTextDelta

If `OnTextDelta` panics or returns an error, the loop MUST continue processing the stream. The error MUST NOT propagate to the stream consumer or terminate the stream.

#### Scenario: OnTextDelta panic contained

- GIVEN an observer whose `OnTextDelta` panics
- WHEN a text delta arrives
- THEN the panic is recovered
- AND the stream continues processing subsequent chunks

#### Scenario: OnTextDelta error does not stop stream

- GIVEN an observer whose `OnTextDelta` returns an error
- WHEN a text delta arrives
- THEN the error is logged/ignored
- AND the stream continues
