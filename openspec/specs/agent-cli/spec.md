# agent-cli Specification

## Purpose

A minimal command-line entry point that wires the core loop with the OpenAI-compatible provider and the built-in tools, proving the loop end-to-end.

## Requirements

### Requirement: REQ-CLI-1 — Run the Loop

The CLI MUST accept a prompt as its argument, run the agent loop, and print the final answer to standard output.

#### Scenario: Prompt with tool use

- GIVEN the prompt argument "list files in ." and a reachable provider
- WHEN the CLI runs
- THEN the loop completes and the final answer is printed to stdout

#### Scenario: No prompt

- GIVEN no prompt argument
- WHEN the CLI starts
- THEN the CLI prints usage text
- AND exits with a non-zero status

### Requirement: REQ-CLI-2 — Failure Reporting

The CLI MUST exit with a non-zero status and print an actionable message to standard error when provider configuration is invalid or the loop fails.

#### Scenario: Missing API key

- GIVEN OPENAI_API_KEY is unset
- WHEN the CLI starts
- THEN an actionable error naming OPENAI_API_KEY is printed to stderr
- AND the exit status is non-zero

#### Scenario: Successful completion

- GIVEN a prompt argument and a provider returning an answer
- WHEN the CLI runs
- THEN the exit status is zero

### Requirement: REQ-CLI-3 — Profile Subcommands

The CLI MUST provide profile subcommands: `list` enumerating resolved profiles and marking the active one, `switch <name>` activating a profile for the session, and `model <name> <model>` setting a persisted per-profile model. Unknown profile names or missing arguments MUST produce a non-zero exit and an actionable stderr message.

#### Scenario: List profiles

- GIVEN two profiles with "coder" active
- WHEN `kui profile list` runs
- THEN both profiles print with "coder" marked active
- AND the exit status is zero

#### Scenario: Switch to unknown profile

- GIVEN `kui profile switch nope` and no such profile
- WHEN the command runs
- THEN stderr names "nope" as unknown
- AND the exit status is non-zero

#### Scenario: Model set persists

- GIVEN `kui profile model coder gpt-4o`
- WHEN the command runs
- THEN the model is persisted for profile "coder"
- AND the exit status is zero

### Requirement: REQ-CLI-4 — Per-Profile Model Resolution

At startup the CLI MUST resolve the model for the active profile in this order: the profile's saved model (from `.kui/`), then the profile.yaml model, then the global default. The resolved model MUST be passed to the provider.

#### Scenario: Saved model wins

- GIVEN profile "coder" with a saved model "gpt-4o" and a profile.yaml model "gpt-4o-mini"
- WHEN the CLI starts
- THEN the provider is configured with "gpt-4o"

#### Scenario: Fallback chain

- GIVEN a profile with no saved model and no profile.yaml model
- WHEN the CLI starts
- THEN the global default model is used

### Requirement: REQ-CLI-5 — TUI Subcommand Dispatch

The CLI MUST provide a `kui tui` subcommand that starts the interactive TUI. When startup validation fails (e.g. invalid provider configuration), `kui tui` MUST exit non-zero with an actionable stderr message. The existing one-shot prompt behavior (REQ-CLI-1) MUST remain unchanged.

#### Scenario: Starts the TUI

- GIVEN valid configuration and a reachable provider
- WHEN `kui tui` runs
- THEN the Bubble Tea app starts

#### Scenario: Startup validation failure

- GIVEN invalid provider configuration
- WHEN `kui tui` runs
- THEN an actionable error prints to stderr
- AND the exit status is non-zero

#### Scenario: One-shot prompt unchanged

- GIVEN a prompt argument and a reachable provider
- WHEN the existing one-shot prompt mode runs
- THEN the loop completes and the final answer prints to stdout
- AND the exit status is zero

### Requirement: REQ-CLI-11 — --model Flag Override

The CLI MUST accept `--model, -m <name>` to override the model resolution chain. When specified, this value MUST take highest priority — above saved model, profile.yaml model, and global default.

#### Scenario: --model overrides saved model

- GIVEN profile "coder" with saved model "gpt-4o" and `--model gpt-4o-mini` on the CLI
- WHEN the CLI starts
- THEN the provider is configured with "gpt-4o-mini"
- AND the saved model "gpt-4o" is ignored

#### Scenario: --model overrides profile default

- GIVEN a profile with profile.yaml model "gpt-4o" and `--model claude-3` on the CLI
- WHEN the CLI starts
- THEN the provider is configured with "claude-3"

#### Scenario: Short flag -m

- GIVEN args `["-m", "gpt-4o", "hello"]`
- WHEN the CLI parses flags
- THEN the resolved model is "gpt-4o"

### Requirement: REQ-CLI-12 — Model Override Flow

The `--model` override value MUST flow through `ResolveModel` and `SetModel` without modifying the persisted profile configuration. The override is session-scoped only.

#### Scenario: Override does not persist

- GIVEN `--model gpt-4o` on the CLI
- WHEN the CLI completes
- THEN the profile's saved model remains unchanged in `.kui/`

#### Scenario: Override used in resolution chain

- GIVEN `--model gpt-4o` and a profile with no saved model
- WHEN `ResolveModel` is called
- THEN "gpt-4o" is returned as the resolved model

### Requirement: REQ-CLI-13 — --model Without Value

The CLI MUST produce an actionable error to stderr and exit non-zero when `--model` or `-m` is specified without a value.

#### Scenario: --model at end of args

- GIVEN args `["--model"]`
- WHEN `parseFlags` is called
- THEN an error is returned indicating missing model value
- AND the exit status is non-zero

#### Scenario: -m without value

- GIVEN args `["-m"]`
- WHEN `parseFlags` is called
- THEN an error is returned indicating missing model value

