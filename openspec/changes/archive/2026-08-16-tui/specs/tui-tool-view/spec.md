# tui-tool-view Specification

## Purpose

The tool view renders live tool calls and results during multi-step turns, driven by the agent loop's observer port.

## Requirements

### Requirement: REQ-TUI-TOOL-1 — Live Tool Events

The tool view MUST render each tool call and its result as the events arrive during a multi-step turn.

#### Scenario: Multi-step turn renders

- GIVEN a turn that invokes "read_file" and then answers
- WHEN the loop emits observer events
- THEN the call "read_file" appears
- AND its result appears once available

### Requirement: REQ-TUI-TOOL-2 — Graceful Degradation

When the observer is nil or unavailable, the tool view MUST stay empty/disabled, MUST NOT crash, and MUST NOT alter the loop's behavior.

#### Scenario: Nil observer

- GIVEN a loop running with a nil observer
- WHEN the app renders
- THEN the tool view shows no events
- AND the app keeps running normally

#### Scenario: Observer unavailable mid-turn

- GIVEN an observer that stops delivering events mid-turn
- WHEN the turn completes
- THEN the tool view degrades without crashing
- AND the loop's termination and output are unaffected
