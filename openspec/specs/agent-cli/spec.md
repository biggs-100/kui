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
