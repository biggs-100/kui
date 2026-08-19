# Plugin Command Dispatch Specification

## Purpose

Wires plugin-registered commands to the TUI command palette. Bridges the existing `RegisterCommand` stub to actual execution via plugin subprocess, enabling plugin-provided commands to appear and execute in the TUI.

## Requirements

### Requirement: REQ-PCMD-1 — Command Registration to Palette

When a plugin calls `RegisterCommand`, the system MUST add the command to the TUI command palette. Commands MUST appear with name, description, and the plugin name as category. Registration MUST happen during plugin Init, before the TUI starts.

#### Scenario: Plugin registers command during Init

- GIVEN a plugin that calls RegisterCommand with name "summarize" and description "Summarize conversation"
- WHEN the plugin Init completes
- THEN "summarize" appears in the TUI command palette
- AND its category shows the plugin name

#### Scenario: Multiple plugins register commands

- GIVEN plugin A registers "cmd-a" and plugin B registers "cmd-b"
- WHEN both plugins are loaded
- THEN both commands appear in the command palette

#### Scenario: Duplicate command name

- GIVEN a built-in command "help" already registered
- WHEN a plugin registers a command named "help"
- THEN the registration is rejected with a duplicate-name error
- AND the built-in command remains

### Requirement: REQ-PCMD-2 — Command Execution

When a user selects a plugin command from the palette, the system MUST execute the command by invoking the plugin subprocess with the command name and arguments. The execution MUST use the existing JSON-RPC 2.0 transport.

#### Scenario: Execute plugin command successfully

- GIVEN a registered plugin command "summarize"
- WHEN the user selects "summarize" from the palette
- THEN the system sends a JSON-RPC request to the plugin subprocess
- AND the command output is displayed in the TUI

#### Scenario: Plugin subprocess not running

- GIVEN a registered plugin command whose subprocess crashed
- WHEN the user selects the command
- THEN an error message is displayed indicating the plugin is unavailable
- AND the TUI remains functional

### Requirement: REQ-PCMD-3 — Command Error Handling

When a plugin command returns an error, the system MUST display the error message in the TUI without crashing. The error MUST include the plugin name and command name for clarity. Repeated failures SHOULD trigger a warning.

#### Scenario: Plugin command returns error

- GIVEN a registered plugin command "summarize"
- WHEN the command execution returns error "model not configured"
- THEN the TUI displays: "plugin:summarize: model not configured"
- AND the TUI remains functional

#### Scenario: Plugin command panics

- GIVEN a registered plugin command that causes a subprocess crash
- WHEN the command is executed
- THEN the error is caught
- AND the TUI displays a crash notification
- AND the plugin is marked unavailable

### Requirement: REQ-PCMD-4 — Command Help Text

Plugin commands MUST support a help text field in the Command struct. Help text MUST be displayed in the command palette as a secondary line or tooltip when the command is highlighted.

#### Scenario: Command with help text

- GIVEN a registered command with help text "Summarize the current conversation into key points"
- WHEN the user highlights the command in the palette
- THEN the help text is displayed below the command description

#### Scenario: Command without help text

- GIVEN a registered command without help text
- WHEN the user highlights the command
- THEN no help text is shown (graceful degradation)

### Requirement: REQ-PCMD-5 — Command Unregistration

When a plugin is shut down or crashes, its commands MUST be removed from the command palette. Commands MUST be re-registered when the plugin is reloaded.

#### Scenario: Plugin shutdown removes commands

- GIVEN plugin "my-plugin" with registered commands "cmd-a" and "cmd-b"
- WHEN the plugin is shut down
- THEN "cmd-a" and "cmd-b" are removed from the command palette

#### Scenario: Plugin reload re-registers commands

- GIVEN a crashed plugin that is reloaded
- WHEN the plugin re-registers its commands during Init
- THEN the commands reappear in the command palette
