# agent-cli Delta — cli-parity

## Purpose

Delta spec adding model-override flag support to the existing agent-cli spec. The `--model` flag inserts at highest priority in the model resolution chain defined by REQ-CLI-4.

## ADDED Requirements

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
