# ADR-0003: OpenAI-compatible chat-completions only (no provider SDKs)

- Status: accepted
- Date: 2026-08-14

## Context

The first kui release must talk to a model backend, but the ecosystem is
fragmented: OpenAI, local servers (Ollama, vLLM, LM Studio), and gateways all
speak variations of the chat-completions protocol. The core defines one
`Provider` port, so the question is which backend protocol the first adapter
implements — and whether to use an official SDK or raw HTTP.

## Decision

Ship exactly one provider adapter, `internal/adapters/providers/openai`,
which speaks the OpenAI-compatible chat-completions protocol over the
standard library's `net/http`:

- POST `{base_url}/chat/completions` with messages, tool definitions, and an
  explicit `model` field (`OPENAI_MODEL`, default `gpt-4o-mini`).
- Base URL overridable via `OPENAI_BASE_URL` so any compatible server —
  including local ones — is reachable.
- Typed error surface: `AuthError` (401), `RateLimitError` (429),
  `ServerError` (5xx), `TransportError`, `ParseError`.
- No third-party SDKs; no streaming in this change.

## Alternatives considered

- **Official OpenAI Go SDK**: convenient but couples the adapter to one
  vendor and one HTTP stack; the protocol layer is thin enough to own.
- **Multiple provider adapters now**: the `Provider` port already allows it,
  but each adapter needs its own fixtures and error mapping; one adapter
  proves the port before multiplying implementations.
- **Custom protocol**: locks out the largest compatible ecosystem and makes
  local testing harder.
- **Streaming first**: adds SSE complexity; the loop currently consumes
  complete responses only.

## Consequences

- Any server exposing a compatible `/chat/completions` endpoint works,
  including fully local ones — good for tests and privacy.
- The typed error surface lets the CLI report failures precisely
  (`provider server error (500)` instead of a raw body dump).
- Non-compatible providers (Anthropic Messages API, etc.) are out of scope
  until a second adapter is added behind the same `core.Provider` port.
- No streaming until a later change needs it.

## Verification notes

- `client_test.go`: httptest fixtures assert method, path, Bearer auth,
  request body (including the `model` field), and the 401/429/5xx/parse
  mapping.
- Default endpoint verified by `TestChatDefaultBaseURL`
  (`https://api.openai.com/v1/chat/completions`).
- Live-API smoke requires `OPENAI_API_KEY` and `OPENAI_MODEL`; CI runs the
  httptest fixtures only.
