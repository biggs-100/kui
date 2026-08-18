# provider-openai-compatible Specification

## Purpose

This adapter speaks the OpenAI-compatible chat-completions protocol over HTTP to any base URL. It is the only provider in this change and relies on environment variables for credentials and endpoint configuration.

## Requirements

### Requirement: REQ-PROV-1 — Chat Completions Request

The provider MUST send a POST request to `{base_url}/chat/completions` carrying the message sequence and tool definitions, and MUST parse the response into messages that may contain tool calls.

#### Scenario: Response with tool call

- GIVEN a compatible server returning a response with a tool call
- WHEN the provider sends the chat request
- THEN the provider returns the tool call to the loop

#### Scenario: Malformed response body

- GIVEN a server returning invalid JSON
- WHEN the provider parses the response
- THEN the provider returns a typed parse error

### Requirement: REQ-PROV-2 — Env Credentials

The provider MUST read the API key from the OPENAI_API_KEY environment variable and MUST fail with an actionable error naming that variable when it is unset or empty. Requests MUST carry the key in the Authorization header.

#### Scenario: Key missing

- GIVEN OPENAI_API_KEY is unset
- WHEN the provider is created
- THEN creation fails with an error naming OPENAI_API_KEY

#### Scenario: Key present

- GIVEN OPENAI_API_KEY is set
- WHEN the provider sends a request
- THEN the request includes the key as a Bearer token

### Requirement: REQ-PROV-3 — Base URL Override

The provider MUST read the base URL from a per-provider environment variable (e.g. `OPENAI_BASE_URL` for openai, `OPENCODE_BASE_URL` for opencode), defaulting to the provider's canonical base URL when unset. The API key MUST NOT appear in any error message.

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

### Requirement: REQ-PROV-4 — Error Surface

The provider MUST map HTTP status codes and transport failures to distinct typed errors: authentication (401), rate limit (429), server error (5xx), and transport failure. Errors MUST NOT contain the API key.

#### Scenario: Authentication failure

- GIVEN a server returning 401
- WHEN the provider receives the response
- THEN the provider returns an authentication error

#### Scenario: Server error

- GIVEN a server returning 500
- WHEN the provider receives the response
- THEN the provider returns a server error

#### Scenario: Transport failure

- GIVEN an unreachable endpoint
- WHEN the provider sends the request
- THEN the provider returns a transport error
