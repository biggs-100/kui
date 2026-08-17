# Delta for thinking-config

## ADDED Requirements

### Requirement: REQ-THINK-1 — Thinking Levels Defined

The system MUST define four thinking levels: `off`, `low`, `medium`, `high`. These levels control how much reasoning effort the model applies before answering.

#### Scenario: All levels accepted

- GIVEN valid thinking levels: off, low, medium, high
- WHEN any level is passed to the resolver
- THEN the level is accepted without error

#### Scenario: Unknown level rejected

- GIVEN a thinking level "banana"
- WHEN passed to the resolver
- THEN an error is returned listing the valid values

### Requirement: REQ-THINK-2 — Default Thinking Level

The system MUST default to `off` (no reasoning effort) when no thinking level is configured anywhere.

#### Scenario: No config, no flag

- GIVEN no profile thinking field and no --thinking flag
- WHEN the thinking level is resolved
- THEN the resolved level is "off"

#### Scenario: Profile sets level, no flag

- GIVEN profile.yaml with `thinking: high` and no --thinking flag
- WHEN the thinking level is resolved
- THEN the resolved level is "high"

### Requirement: REQ-THINK-3 — Per-Profile Thinking Field

A profile.yaml MUST accept a `thinking` field with a valid level string. The loader MUST parse this field alongside existing profile fields.

#### Scenario: Valid thinking field in profile

- GIVEN profile.yaml with `thinking: medium`
- WHEN the loader parses it
- THEN the profile's Thinking field is "medium"

#### Scenario: Missing thinking field

- GIVEN profile.yaml with no `thinking` field
- WHEN the loader parses it
- THEN the profile's Thinking field is empty (resolved to "off" by default)

### Requirement: REQ-THINK-4 — Layered Thinking Resolution

Thinking level MUST resolve through the layered config chain: global → project → profile, with the nearest layer winning. CLI `--thinking` overrides all layers.

#### Scenario: Profile overrides project

- GIVEN project config with `thinking: low` and profile with `thinking: high`
- WHEN resolved for that profile
- THEN the result is "high"

#### Scenario: Global fallback

- GIVEN no project or profile thinking setting, global with `thinking: medium`
- WHEN resolved
- THEN the result is "medium"

#### Scenario: CLI overrides profile

- GIVEN profile with `thinking: low` and `--thinking high` on CLI
- WHEN resolved
- THEN the result is "high"
