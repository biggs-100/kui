# Delta for agent-loop

## ADDED Requirements

### Requirement: REQ-LOOP-7 — Observer Port (nil-safe, stdlib)

The loop MUST expose an optional observer port through which tool-call, tool-result, and turn events are published. The port MUST use stdlib types only. When the observer is nil, the loop MUST behave identically to today — all existing loop tests MUST pass unmodified. Observer events MUST NOT influence loop control flow: termination, iteration budget, and event ordering MUST be unchanged regardless of observer behavior or absence.

#### Scenario: Nil observer is a no-op

- GIVEN a loop constructed with a nil observer
- WHEN the loop runs any existing scenario
- THEN behavior is identical to a loop without an observer
- AND all existing loop tests pass unmodified

#### Scenario: Tool events published

- GIVEN a loop with an observer attached
- WHEN a tool call and its result occur
- THEN the observer receives the call event
- AND the result event

#### Scenario: Turn events published

- GIVEN a loop with an observer attached
- WHEN a turn starts and completes
- THEN the observer receives start and completion events

#### Scenario: Observer failure is contained

- GIVEN an observer that panics or errors on an event
- WHEN the loop emits an event
- THEN the panic is contained and the loop continues unaffected
