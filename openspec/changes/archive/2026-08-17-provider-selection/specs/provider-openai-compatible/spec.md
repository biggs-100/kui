# Delta for provider-openai-compatible

## MODIFIED Requirements

### Requirement: REQ-PROV-3 — Base URL Override

The provider MUST read the base URL from a per-provider environment variable (e.g. `OPENAI_BASE_URL` for openai, `OPENCODE_BASE_URL` for opencode), defaulting to the provider's canonical base URL when unset. The API key MUST NOT appear in any error message.

(Previously: Only read `OPENAI_BASE_URL` env var with a single hardcoded default)

#### Scenario: Custom base URL

- GIVEN OPENAI_BASE_URL pointing at a local httptest server
- WHEN the provider sends a request
- THEN the request targets the custom base URL

#### Scenario: Default base URL

- GIVEN OPENAI_BASE_URL is unset
- WHEN the provider sends a request
- THEN the request targets the provider's canonical base URL

#### Scenario: Per-provider base URL env var

- GIVEN OPENCODE_BASE_URL set to `http://localhost:8080/v1`
- WHEN the opencode provider sends a request
- THEN the request targets `http://localhost:8080/v1/chat/completions`
