# steering-followup Specification

## Purpose

The loop carries two pending-message queues with different drain semantics. The steering queue drains after every turn; the follow-up queue drains only when the agent would otherwise stop. Each queue has a mode: all or one-at-a-time. The queues are additive — they extend the turn sequence without changing the termination contract.

## Requirements

### Requirement: REQ-QUEUE-1 — Steering Drain

The steering queue MUST be drained after each completed turn: queued messages MUST be injected before the next provider request. In all mode, the entire queue drains in order per turn; in one-at-a-time mode, exactly one message drains per turn.

#### Scenario: Drain all

- GIVEN three queued steering messages and mode all
- WHEN the next turn begins
- THEN all three messages are injected before the provider request

#### Scenario: Drain one per turn

- GIVEN three queued steering messages and mode one-at-a-time
- WHEN three turns run
- THEN exactly one message is injected per turn, in order

### Requirement: REQ-QUEUE-2 — Follow-Up Drain

The follow-up queue MUST be drained only when the agent would otherwise stop: after the provider returns no tool calls and the steering queue is empty. Drained messages MUST be injected as new turns.

#### Scenario: Drains at stop

- GIVEN a queued follow-up message and a provider ready to answer
- WHEN the provider returns without tool calls
- THEN the follow-up message is injected
- AND the loop continues instead of stopping

#### Scenario: Empty follow-up queue

- GIVEN an empty follow-up queue
- WHEN the provider returns without tool calls
- THEN the loop stops normally

### Requirement: REQ-QUEUE-3 — Termination Contract

The queues MUST NOT alter the REQ-LOOP-3 termination rules: iteration-budget and tool-failure termination MUST apply with queued messages present. Queued messages MUST NOT be treated as tool calls.

#### Scenario: Budget with queues

- GIVEN a 3-iteration budget and follow-up messages that keep the loop alive
- WHEN the budget is exhausted
- THEN the loop terminates with the iteration-limit error
- AND no further provider requests are made

#### Scenario: Tool failure with queued steering

- GIVEN a queued steering message and a failing tool
- WHEN the tool fails
- THEN the loop terminates with the tool's error
- AND the queued message is not injected
