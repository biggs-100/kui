# agent-loop Specification

## Purpose

The agent loop is the domain core of kui: it orchestrates a single-session conversation between an LLM provider and local tools, from a user prompt to a final answer or a bounded-termination error. It performs no I/O and depends only on its own ports.

## Requirements

### Requirement: REQ-LOOP-1 — Loop Execution

The loop MUST accept a user prompt, send it to the provider port, and return the provider's final answer once the provider emits a response without tool calls.

#### Scenario: Direct answer without tools

- GIVEN a provider configured to answer without tool calls
- WHEN the loop runs with the prompt "hello"
- THEN the loop returns the provider's answer content

#### Scenario: Multi-step tool resolution

- GIVEN a provider that first requests a tool call and then answers
- WHEN the loop runs with a prompt requiring that tool
- THEN the loop dispatches the tool, feeds the result back to the provider, and returns the final answer

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
