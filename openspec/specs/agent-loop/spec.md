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

(Previously: Observer port for read-only event publication; no hook registry mentioned.)

The loop MUST expose an optional observer port through which tool-call, tool-result, and turn events are published. The port MUST use stdlib types only. When the observer is nil, the loop MUST behave identically to today — all existing loop tests MUST pass unmodified. Observer events MUST NOT influence loop control flow: termination, iteration budget, and event ordering MUST be unchanged regardless of observer behavior or absence. The loop MUST ALSO accept an optional `*HookRegistry` field. When non-nil, the HookRegistry is emitted at defined lifecycle points. The HookRegistry is a mutable counterpart to the read-only Observer — hooks can modify messages, block tool execution, and alter provider requests.

#### Scenario: Nil observer is a no-op

- GIVEN a loop constructed with a nil observer and nil HookRegistry
- WHEN the loop runs any existing scenario
- THEN behavior is identical to a loop without an observer or hooks
- AND all existing loop tests pass unmodified

#### Scenario: Tool events published via Observer

- GIVEN a loop with an observer attached
- WHEN a tool call and its result occur
- THEN the observer receives the call event
- AND the result event

#### Scenario: Turn events published via Observer

- GIVEN a loop with an observer attached
- WHEN a turn starts and completes
- THEN the observer receives start and completion events

#### Scenario: Observer failure is contained

- GIVEN an observer that panics or errors on an event
- WHEN the loop emits an event
- THEN the panic is contained and the loop continues unaffected

### Requirement: REQ-LOOP-12 — HookRegistry Integration

The loop MUST accept an optional `*HookRegistry` field. When nil, the loop behaves exactly as before — no hooks fire, no HookContext is allocated, and all existing tests pass unchanged. When non-nil, the registry is emitted at the lifecycle points defined by REQ-LOOP-13 through REQ-LOOP-15.

#### Scenario: Nil HookRegistry — backward compatible

- GIVEN a loop with nil HookRegistry
- WHEN the loop runs any existing scenario
- THEN no hooks fire
- AND all existing loop tests pass unmodified

#### Scenario: Non-nil HookRegistry — hooks fire

- GIVEN a loop with a HookRegistry containing registered handlers
- WHEN the loop runs
- Then registered hooks fire at the defined lifecycle points

### Requirement: REQ-LOOP-13 — before_provider_request Hook

Before each LLM provider request, the loop MUST emit `before_provider_request` via the HookRegistry. The HookContext MUST expose the current message slice via `Messages()`. Handlers MAY call `SetMessages()` to modify the messages sent to the provider. The modified messages MUST be used for the provider request.

#### Scenario: Hook modifies messages before provider call

- GIVEN a HookRegistry with a handler for "before_provider_request"
- WHEN the loop is about to call the provider
- Then the handler receives a HookContext with current messages
- AND the handler calls SetMessages to prepend a system instruction
- AND the provider receives the modified message set

#### Scenario: Hook error does not abort the loop

- GIVEN a handler for "before_provider_request" that returns an error
- WHEN the hook is emitted
- Then the error is logged/observed
- AND the loop continues with the original (unmodified) messages

### Requirement: REQ-LOOP-14 — before_tool_execution Hook

Before executing any tool, the loop MUST emit `before_tool_execution` via the HookRegistry. The HookContext MUST expose the pending ToolCall via `ToolCall()`. Handlers MAY call `Block(reason)` to prevent execution. If `IsBlocked()` is true after hooks complete, the loop MUST skip tool execution and return a blocked-tool result to the provider.

#### Scenario: Hook blocks tool execution

- GIVEN a HookRegistry with a handler that calls Block("policy") for tool "bash"
- WHEN the loop is about to execute "bash"
- Then the handler receives the ToolCall in HookContext
- AND the handler calls Block("policy")
- AND the tool is NOT executed
- AND the loop returns a blocked-tool result to the provider

#### Scenario: Hook allows tool execution

- GIVEN a HookRegistry with a handler that does NOT call Block
- WHEN the loop is about to execute a tool
- Then the handler runs successfully
- AND the tool executes normally

### Requirement: REQ-LOOP-15 — after_tool_execution Hook

After tool execution completes, the loop MUST emit `after_tool_execution` via the HookRegistry. The HookContext MUST expose the tool result. This hook is read-only for result observation — handlers MUST NOT be able to modify the result via this hook.

#### Scenario: Hook observes tool result

- GIVEN a HookRegistry with a handler for "after_tool_execution"
- WHEN a tool returns a result
- Then the handler receives the HookContext with the tool result
- AND the result is returned to the provider unmodified

#### Scenario: Hook error does not corrupt result

- GIVEN a handler for "after_tool_execution" that returns an error
- WHEN the tool completes
- Then the error is logged/observed
- AND the tool result is still returned to the provider correctly

