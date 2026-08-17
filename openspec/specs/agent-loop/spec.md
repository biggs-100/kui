# agent-loop Specification

## Purpose

The agent loop is the domain core of kui: it orchestrates a single-session conversation between an LLM provider and local tools, from a user prompt to a final answer or a bounded-termination error. It performs no I/O and depends only on its own ports.

## Requirements

### Requirement: REQ-LOOP-1 — Loop Execution

The loop MUST accept a user prompt, send it to the provider port, and return the provider's final answer once the provider emits a response without tool calls. The loop MUST operate in two levels: an inner level that alternates provider requests with tool execution and steering-queue draining, and an outer level that drains the follow-up queue only when the inner level would otherwise stop. Queued messages MUST be additive: they extend the turn sequence without changing the termination contract.
(Previously: single-level loop with no pending-message queues.)

#### Scenario: Direct answer without tools

- GIVEN a provider configured to answer without tool calls
- WHEN the loop runs with the prompt "hello"
- THEN the loop returns the provider's answer content

#### Scenario: Multi-step tool resolution

- GIVEN a provider that first requests a tool call and then answers
- WHEN the loop runs with a prompt requiring that tool
- THEN the loop dispatches the tool, feeds the result back to the provider, and returns the final answer

#### Scenario: Steering message between turns

- GIVEN a queued steering message
- WHEN a turn completes and a new provider request is about to be sent
- THEN the queued message is injected before that request

#### Scenario: Termination with empty queues

- GIVEN empty steering and follow-up queues
- WHEN the provider returns no tool calls
- THEN the loop terminates normally, as before the restructure

### Requirement: REQ-LOOP-2 — Tool Contract

The loop MUST communicate with tools exclusively through the tool port: a name, a description, a JSON parameter schema, and an Execute(input) operation. The loop MUST NOT depend on concrete tool implementations.

#### Scenario: Dispatch through the port

- GIVEN a registered tool named "read_file"
- WHEN the provider requests that tool
- THEN the loop invokes it through the port with the requested input

#### Scenario: Unknown tool request

- GIVEN a provider requesting a tool that is not registered
- WHEN the loop processes the response
- THEN the loop terminates with a typed unknown-tool error
- AND no further provider requests are made

### Requirement: REQ-LOOP-3 — Termination Rules

The loop MUST support a configurable maximum iteration count and MUST terminate when the provider returns no tool calls, when the iteration budget is exhausted, or when a tool execution fails.

#### Scenario: Iteration budget exhausted

- GIVEN a maximum of 3 iterations and a provider that keeps requesting tool calls
- WHEN the loop runs
- THEN the loop terminates after 3 iterations with an iteration-limit error
- AND no further provider requests are made

#### Scenario: Tool execution failure

- GIVEN a registered tool whose Execute operation fails
- WHEN the loop dispatches it
- THEN the loop terminates with the tool's error
- AND the error identifies the failing tool

### Requirement: REQ-LOOP-4 — Provider Contract

The loop MUST communicate with the model exclusively through the provider port, exchanging message sequences that may include tool calls and tool results.

#### Scenario: Tool result returned to provider

- GIVEN a tool that returned a successful result
- WHEN the loop feeds the result back
- THEN the provider receives a message containing the tool result and its tool-call identifier

### Requirement: REQ-LOOP-5 — Profile Switch Between Turns

A queued profile switch MUST apply between turns — never during a tool call and never mid-response. The switch MUST NOT alter the conversation history.

#### Scenario: Switch applies between turns

- GIVEN a switch message queued during a tool call
- WHEN the tool call completes
- THEN the active profile changes before the next provider request
- AND the history is unchanged

#### Scenario: Multiple switches in queue

- GIVEN two switch messages drained in the same steering drain
- WHEN the next provider request is built
- THEN the last switch determines the active profile

### Requirement: REQ-LOOP-6 — Profile-Context Marker

When a profile switch applies, the loop MUST insert a profile-context marker message into the history before the next provider request, identifying the newly active profile.

#### Scenario: Marker on switch

- GIVEN a successful switch to profile "coder"
- WHEN the next turn begins
- THEN the history contains a marker naming "coder"

#### Scenario: No marker without switch

- GIVEN a session with no profile switch
- WHEN turns proceed
- THEN no marker messages are inserted

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

