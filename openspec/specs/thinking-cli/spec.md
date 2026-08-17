# Delta for thinking-cli

## MODIFIED Requirements

### Requirement: REQ-CLI-10 — Options Struct

The system MUST define an `Options` struct holding all parsed flag values: `Model string`, `Tools string`, `ExcludeTools string`, `NoTools bool`, `NoExtensions bool`, `NoSkills bool`, `NoSession bool`, `Verbose bool`, `Mode string`, `Approve bool`, `Print bool`, `Thinking string`.

(Previously: Options struct did not include Thinking field)

#### Scenario: All fields default to zero values

- GIVEN no flags provided
- WHEN `parseFlags` is called with empty args
- THEN all Options fields are their zero values (empty string or false)

#### Scenario: Partial flags set

- GIVEN args `["--verbose", "--model", "gpt-4o"]`
- WHEN `parseFlags` is called
- THEN `Options.Verbose == true` and `Options.Model == "gpt-4o"`
- AND other fields remain at zero values

#### Scenario: Thinking flag set

- GIVEN args `["--thinking", "high"]`
- WHEN `parseFlags` is called
- THEN `Options.Thinking == "high"`

## ADDED Requirements

### Requirement: REQ-THINK-5 — --thinking Flag

The parser MUST accept `--thinking <level>` where level is one of: off, low, medium, high. The flag value MUST be stored in `Options.Thinking`.

#### Scenario: --thinking with space separator

- GIVEN args `["--thinking", "medium"]`
- WHEN `parseFlags` is called
- THEN `Options.Thinking == "medium"`

#### Scenario: --thinking with equals

- GIVEN args `["--thinking=high"]`
- WHEN `parseFlags` is called
- THEN `Options.Thinking == "high"`

### Requirement: REQ-THINK-6 — Thinking Flag Priority

The `--thinking` flag MUST have the highest priority in the resolution chain, overriding profile, project, and global config.

#### Scenario: CLI overrides profile

- GIVEN profile.yaml with `thinking: low` and `--thinking high` on CLI
- WHEN the thinking level is resolved
- THEN the resolved level is "high"

#### Scenario: CLI overrides global

- GIVEN global config with `thinking: medium` and `--thinking off` on CLI
- WHEN the thinking level is resolved
- THEN the resolved level is "off"

### Requirement: REQ-THINK-7 — Invalid Thinking Level Error

The parser MUST return an actionable error when an invalid thinking level is provided, listing the valid values.

#### Scenario: Invalid level

- GIVEN args `["--thinking", "banana"]`
- WHEN `parseFlags` is called
- THEN an error is returned containing "banana" and listing valid values

#### Scenario: Empty level

- GIVEN args `["--thinking"]` with no value
- WHEN `parseFlags` is called
- THEN an error is returned indicating a value is required

### Requirement: REQ-THINK-8 — Profile Thinking Subcommand

The CLI MUST support `kui profile thinking <name> <level>` to persist a thinking level for a named profile.

#### Scenario: Set thinking for profile

- GIVEN a profile named "coder" exists
- WHEN `kui profile thinking coder high` is executed
- THEN the profile.yaml for "coder" is updated with `thinking: high`

#### Scenario: Invalid level for profile subcommand

- GIVEN a profile named "coder" exists
- WHEN `kui profile thinking coder banana` is executed
- THEN an error is returned listing valid values
