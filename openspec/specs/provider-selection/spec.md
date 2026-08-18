# provider-selection Specification

## Purpose

Provider registry and selection system that maps provider names to adapters, resolves API keys, validates configuration at startup, and surfaces thinking-capability mismatches.

## Requirements

### Requirement: REQ-SEL-1 — Provider Registry

The system MUST maintain a registry mapping provider names (e.g. `openai`, `opencode`) to adapter constructors. Each registered provider MUST declare its required environment variable and supported capabilities (e.g. thinking support).

#### Scenario: Known provider lookup

- GIVEN a registry containing `openai`
- WHEN `resolve("openai")` is called
- THEN the openai adapter constructor is returned

#### Scenario: Unknown provider lookup

- GIVEN a registry not containing `anthropic`
- WHEN `resolve("anthropic")` is called
- THEN an actionable error naming `anthropic` is returned

### Requirement: REQ-SEL-2 — Layered Provider Resolution

Provider selection MUST follow the same layered chain as model resolution: `--provider` flag > `provider:` field in profile.yaml > environment variable (e.g. `KUI_PROVIDER`) > default (`openai`).

#### Scenario: Flag overrides profile

- GIVEN profile.yaml declares `provider: opencode` and `--provider openai` is passed
- WHEN resolution runs
- THEN provider is `openai`

#### Scenario: Default fallback

- GIVEN no flag, no profile provider field, no env var
- WHEN resolution runs
- THEN provider is `openai`

### Requirement: REQ-SEL-3 — Fail-Fast API Key Validation

At provider creation time, the system MUST verify the required API key environment variable is set and non-empty. If missing, creation MUST fail immediately with an error naming the variable.

#### Scenario: Key present

- GIVEN `OPENAI_API_KEY` is set
- WHEN the openai provider is created
- THEN creation succeeds

#### Scenario: Key missing

- GIVEN `OPENAI_API_KEY` is unset
- WHEN the openai provider is created
- THEN creation fails with an error naming `OPENAI_API_KEY`

### Requirement: REQ-SEL-4 — Thinking Degradation Warning

When a provider is selected and thinking mode is active, the system MUST check the provider's thinking capability. If the provider lacks thinking support, the system MUST emit a warning and continue without thinking — never error.

#### Scenario: Provider supports thinking

- GIVEN provider with thinking capability and thinking enabled
- WHEN provider is created
- THEN no warning is emitted

#### Scenario: Provider lacks thinking

- GIVEN provider without thinking capability and thinking enabled
- WHEN provider is created
- THEN a warning is emitted
- AND provider creation succeeds
