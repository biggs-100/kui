# output-verbosity Specification

## Purpose

CLI flags controlling output format, verbosity, and non-interactive behavior. Enables scripting (`--mode json`), debugging (`--verbose`), one-shot execution (`--print`), and permission bypass (`--approve`).

## Requirements

### Requirement: REQ-CLI-22 — --verbose Debug Output

The CLI MUST accept `--verbose` as a boolean flag. When set, debug information (loop turns, tool calls, provider requests) MUST be written to stderr. stdout MUST contain only the final answer.

#### Scenario: Verbose writes to stderr

- GIVEN `--verbose` and a prompt "hello"
- WHEN the CLI runs
- THEN debug info appears on stderr
- AND the final answer appears on stdout only

#### Scenario: Default is quiet

- GIVEN no `--verbose` flag
- WHEN the CLI runs
- THEN no debug info is written to stderr

### Requirement: REQ-CLI-23 — --mode json Output

The CLI MUST accept `--mode json` to wrap the final answer in a JSON envelope: `{"answer":"..."}`. The JSON MUST be written to stdout. `--mode json` MUST be rejected with a non-zero exit when combined with the `tui` subcommand.

#### Scenario: JSON output

- GIVEN `--mode json` and a prompt "hello"
- WHEN the CLI runs
- THEN stdout contains `{"answer":"<the answer>"}`

#### Scenario: JSON rejected with TUI

- GIVEN `--mode json` and the `tui` subcommand
- WHEN the CLI starts
- THEN an error is printed to stderr
- AND the exit status is non-zero

### Requirement: REQ-CLI-24 — --mode text Default

The CLI MUST accept `--mode text` (or omit `--mode`) for existing default behavior: plain text answer printed to stdout.

#### Scenario: Explicit text mode

- GIVEN `--mode text` and a prompt "hello"
- WHEN the CLI runs
- THEN stdout contains the plain text answer

#### Scenario: Default behavior unchanged

- GIVEN no `--mode` flag
- WHEN the CLI runs
- THEN behavior is identical to `--mode text`

### Requirement: REQ-CLI-25 — --print Alias

The CLI MUST accept `--print, -p` as a boolean flag. This is an alias for the existing non-interactive one-shot behavior (REQ-CLI-1). It documents the current default for scripting clarity.

#### Scenario: --print flag

- GIVEN `--print` and a prompt "hello"
- WHEN the CLI runs
- THEN the loop completes and the answer prints to stdout

#### Scenario: Short flag -p

- GIVEN args `["-p", "hello"]`
- WHEN the CLI parses flags
- THEN non-interactive mode is active

### Requirement: REQ-CLI-26 — --approve Permission Bypass

The CLI MUST accept `--approve, -a` as a boolean flag. When set, the CLI MUST bypass all permission prompts and tool approval rules — every tool call is permitted. A warning MUST be written to stderr documenting that this bypasses security.

#### Scenario: Approve bypasses permissions

- GIVEN `--approve` and a tool that normally requires approval
- WHEN the loop dispatches that tool
- THEN the tool executes without prompting

#### Scenario: Warning emitted

- GIVEN `--approve`
- WHEN the CLI starts
- THEN a warning is written to stderr about permission bypass

#### Scenario: Short flag -a

- GIVEN args `["-a", "hello"]`
- WHEN the CLI parses flags
- THEN approval is bypassed
