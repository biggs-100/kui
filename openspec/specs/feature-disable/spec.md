# feature-disable Specification

## Purpose

CLI flags that skip optional subsystem initialization — extensions, skills, and session — for faster one-shot invocations and scripting use cases.

## Requirements

### Requirement: REQ-CLI-19 — --no-extensions Skip

The CLI MUST accept `--no-extensions, -ne` as a boolean flag. When set, the CLI MUST skip the `extensions.LoadAll()` call entirely. The agent loop MUST run without extension-loaded tools or hooks.

#### Scenario: Extensions skipped

- GIVEN `--no-extensions` and a `.kui/extensions/` directory with installed extensions
- WHEN the CLI starts
- THEN `extensions.LoadAll()` is NOT called
- AND the loop runs with only built-in tools

#### Scenario: Short flag -ne

- GIVEN args `["-ne", "hello"]`
- WHEN the CLI parses flags
- THEN extensions are not loaded

### Requirement: REQ-CLI-20 — --no-skills Skip

The CLI MUST accept `--no-skills, -ns` as a boolean flag. When set, the CLI MUST pass a nil skills index to the agent. Skills are not resolved, not injected into the system prompt, and not available as tool sources.

#### Scenario: Skills index not built

- GIVEN `--no-skills` and a profile with configured skills
- WHEN the CLI starts
- THEN `skills.NewIndex()` is NOT called
- AND the agent receives a nil skills index

#### Scenario: Short flag -ns

- GIVEN args `["-ns", "hello"]`
- WHEN the CLI parses flags
- THEN skills are not loaded

### Requirement: REQ-CLI-21 — --no-session Placeholder

The CLI MUST accept `--no-session` as a boolean flag. This flag is currently a no-op but documents the intent that kui has no session persistence yet. The flag MUST be accepted without error and MUST NOT affect runtime behavior.

#### Scenario: Flag accepted without error

- GIVEN args `["--no-session", "hello"]`
- WHEN `parseFlags` is called
- THEN `Options.NoSession == true`
- AND no error is returned

#### Scenario: No behavioral change

- GIVEN `--no-session` set
- WHEN the CLI runs
- THEN behavior is identical to running without the flag
