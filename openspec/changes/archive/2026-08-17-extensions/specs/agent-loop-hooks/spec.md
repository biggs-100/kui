# agent-loop-hooks — Delta for agent-loop

## MODIFIED Requirements

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

## ADDED Requirements

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
