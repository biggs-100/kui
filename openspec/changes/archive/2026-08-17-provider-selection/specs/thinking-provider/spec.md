# Delta for thinking-provider

## ADDED Requirements

### Requirement: REQ-THINK-13 — Thinking Capability Check

When the runtime creates a provider with thinking enabled, it MUST check whether the provider advertises thinking support. If unsupported, the system MUST emit a warning and degrade gracefully — omitting `reasoning_effort` from requests rather than failing.

#### Scenario: Provider supports thinking

- GIVEN a provider with thinking capability and thinking level "high"
- WHEN the provider is created
- THEN `reasoning_effort` is included in requests

#### Scenario: Provider lacks thinking

- GIVEN a provider without thinking capability and thinking level "high"
- WHEN the provider is created
- THEN a warning is emitted
- AND `reasoning_effort` is omitted from all requests
