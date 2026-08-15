# Delta for agent-cli

## ADDED Requirements

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
