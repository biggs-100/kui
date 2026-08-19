# Session CLI Specification

## Purpose

CLI subcommands for listing and resuming sessions from the terminal.

## Requirements

### Requirement: Session List Command

`kui session list` SHALL display all saved sessions in a table format showing: ID, profile, model, provider, created timestamp, and message count. Sessions SHALL be sorted by `created_at` descending (most recent first).

#### Scenario: List sessions with data

- GIVEN 3 saved sessions exist
- WHEN `kui session list` is executed
- THEN a table is printed with 3 rows
- AND each row shows id, profile, model, provider, timestamp, and message count
- AND sessions are ordered newest first

#### Scenario: List sessions when none exist

- GIVEN no sessions have been saved
- WHEN `kui session list` is executed
- THEN the output says "No sessions found"

### Requirement: Session Resume Command

`kui session resume <id>` SHALL load the session with the given ID and launch the TUI with the full conversation history injected into the agent context.

#### Scenario: Resume existing session

- GIVEN a session with ID `coder-2026-08-18-1015` exists
- WHEN `kui session resume coder-2026-08-18-1015` is executed
- THEN the TUI starts with the session's message history loaded
- AND new messages are appended to the same session file

#### Scenario: Resume non-existent session

- GIVEN no session with ID `missing-id` exists
- WHEN `kui session resume missing-id` is executed
- THEN an error message is printed: "Session not found: missing-id"
- AND the exit code is non-zero

### Requirement: Default Fresh Behavior

`kui tui` without arguments SHALL start a new empty session. Session resume is only triggered via `kui session resume <id>` or the `/resume` TUI command.

#### Scenario: Fresh start

- GIVEN the user runs `kui tui`
- WHEN the TUI launches
- THEN a new empty session is created
- AND no previous history is loaded
