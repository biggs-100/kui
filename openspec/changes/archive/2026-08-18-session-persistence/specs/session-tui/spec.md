# Session TUI Specification

## Purpose

TUI integration for session lifecycle: auto-save on exit, restore on resume, and inline session management commands.

## Requirements

### Requirement: Auto-Save on Exit

The TUI SHALL save the current session automatically when the user quits (Ctrl+C, `/quit`, or `/exit`). The session file SHALL be written to `.kui/sessions/{id}.json` with all messages and metadata.

#### Scenario: Save on quit

- GIVEN a conversation with messages in the TUI
- WHEN the user quits via `/quit`
- THEN the session is saved to `.kui/sessions/{id}.json`
- AND the index is updated

#### Scenario: Save on interrupt

- GIVEN a conversation with messages in the TUI
- WHEN the user presses Ctrl+C
- THEN the session is saved before the process exits

### Requirement: Auto-Save After Response

The TUI SHALL save the session after every prompt response completes. This ensures crash resilience without requiring explicit save commands.

#### Scenario: Save after each response

- GIVEN the user sends a prompt
- WHEN the assistant response completes
- THEN the session file is updated with the new messages
- AND the `updated_at` timestamp is refreshed

### Requirement: Session List Command

`/sessions` SHALL display saved sessions inline in the TUI, showing ID, profile, model, timestamp, and message count. Output SHALL be formatted for terminal readability.

#### Scenario: List sessions inline

- GIVEN saved sessions exist
- WHEN the user types `/sessions`
- THEN a formatted list is displayed in the TUI
- AND the user can continue chatting after viewing

### Requirement: Session Resume Command

`/resume <id>` SHALL load the specified session's history and replace the current conversation context. The session ID becomes the active session for subsequent saves.

#### Scenario: Resume session

- GIVEN session `coder-2026-08-18-1015` exists
- WHEN the user types `/resume coder-2026-08-18-1015`
- THEN the session history is loaded into the conversation
- AND subsequent messages are saved to that session

#### Scenario: Resume with invalid ID

- GIVEN no session with ID `bad-id` exists
- WHEN the user types `/resume bad-id`
- THEN an error message is shown: "Session not found: bad-id"
- AND the current conversation is unchanged
