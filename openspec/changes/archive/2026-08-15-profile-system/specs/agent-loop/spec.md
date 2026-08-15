# Delta for agent-loop

## MODIFIED Requirements

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

## ADDED Requirements

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
