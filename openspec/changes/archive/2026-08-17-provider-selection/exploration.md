# Exploration: Provider Selection

## Current State

kui ships exactly **one provider adapter** — `internal/adapters/providers/openai/` — which speaks the OpenAI-compatible chat-completions protocol over `net/http` (ADR-0003). Provider creation is hardcoded to `openai.NewClient()` in both `cmd/kui/main.go` (runPrompt, line 380) and `internal/tui/run.go` (Run, line 95–98). The `runtime.Config.Client` factory (`func() (core.Provider, error)`) is the composition injection point, but the concrete function always returns an OpenAI client.

The `core.Provider` interface (38 lines) is clean and generic — it knows nothing about providers:

```go
type Provider interface {
    Chat(ctx context.Context, messages []Message, tools []Tool) ([]Message, error)
}
```

`core.StreamingProvider` extends it opt-in via type assertion. The `openai.Client` also satisfies `setModeler` (SetModel) and `SetThinking` — these are concrete-type methods, not part of the interface.

The profile system already has a `model` field in `profile.yaml` (3-layer resolution: global → project → profile). The `--model` flag overrides everything. But there is **no `provider` field** anywhere — not in profile.yaml, not in the Options struct, not in env vars beyond `OPENAI_*`.

ADR-0003 explicitly deferred this: "Non-compatible providers (Anthropic Messages API, etc.) are out of scope until a second adapter is added behind the same `core.Provider` port."

## Affected Areas

- `internal/adapters/providers/openai/client.go` — current sole provider; env vars are `OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_MODEL`
- `internal/adapters/providers/` — new subdirectories for each added provider (e.g., `anthropic/`)
- `cmd/kui/main.go` — hardcoded `openai.NewClient()` at lines 380, 596; provider creation must become dynamic
- `cmd/kui/flags.go` — Options struct needs `Provider string` field; parseFlags needs `--provider`/`-p`
- `internal/tui/run.go` — Wiring.Client factory hardcoded to `openai.NewClient()` (line 596)
- `internal/adapters/profile/loader.go` — profile.yaml Config struct needs `Provider string` field
- `internal/core/provider.go` — Provider interface is fine; no change needed
- `internal/runtime/runtime.go` — Config.Client factory must receive provider name
- `internal/agent/model.go` — resolution chain may need provider-awareness (thinking levels are provider-specific)

## Approaches

### Approach 1: Provider Registry + Factory Function

Create a provider registry that maps names to factory functions. The `--provider` flag and profile.yaml `provider` field feed into a resolution chain that picks the name, then the registry produces the concrete client.

```
Registry: map[string]func() (core.Provider, error)
  "openai"    → openai.NewClient
  "anthropic" → anthropic.NewClient

Resolution chain: --provider flag → profile.yaml provider → env KUI_PROVIDER → default "openai"
```

**Pros**: Clean separation; new providers are just a new package + one registry entry; follows existing hexagonal pattern; the registry lives in composition root (cmd/kui or runtime), not in core.

**Cons**: Each provider needs its own env var handling, error mapping, and feature set. Thinking/extended-thinking semantics differ across providers (OpenAI: `reasoning_effort`, Anthropic: `extended_thinking` with `budget_tokens`).

**Effort**: Medium — the registry is trivial, but the Anthropic adapter is a real piece of work (different wire format, different auth header, different streaming SSE shape).

### Approach 2: Provider Alias via OpenAI-Compatible Proxy

Add `--provider` as a routing hint only — if "anthropic" is selected, set `OPENAI_BASE_URL` to an OpenAI-compatible proxy (like LiteLLM or a hypothetical kui proxy) and keep using the same OpenAI adapter.

**Pros**: Zero new adapter code; single implementation path.

**Cons**: External dependency on a proxy; loses native Anthropic features (extended thinking, prompt caching, vision); "provider selection" is fake — it's really "proxy selection." Doesn't match Pi's approach.

**Effort**: Low — but deceptive. The proxy becomes a hidden dependency.

### Approach 3: Interface Segregation with Provider Capabilities

Extend the port surface with capability interfaces (like `ThinkingProvider`, `StreamingProvider` already does) so each provider declares what it supports. The CLI/TUI can warn or degrade gracefully when a feature isn't available.

```go
type ThinkingProvider interface {
    SetThinking(level string)
}
type ProviderCapabilities interface {
    Capabilities() []string // "streaming", "thinking", "vision", ...
}
```

**Pros**: Runtime feature detection; graceful degradation; future-proof for tool-calling differences.

**Cons**: More interfaces to maintain; capability checking adds complexity to every feature path.

**Effort**: High — adds a capability system on top of the registry.

## Recommendation

**Approach 1 (Provider Registry + Factory)** with a lightweight **capability hint** from Approach 3.

The registry is the minimum viable structure. Add a simple `Capabilities() []string` method to the concrete providers (not the interface) so the CLI can warn: `"--thinking has no effect with provider=anthropic (use extended_thinking in profile.yaml)"`. This avoids the full capability system while giving users actionable feedback.

The resolution chain follows the exact same pattern as the existing model resolution (REQ-CLI-4):

```
--provider flag (highest) → profile.yaml provider → KUI_PROVIDER env → default "openai"
```

For the first implementation, ship:
1. Provider registry in `cmd/kui` (or `internal/runtime`)
2. `--provider` / `-pr` flag
3. `provider:` field in profile.yaml
4. Anthropic adapter (`internal/adapters/providers/anthropic/`) — Messages API with tool_use, extended thinking support
5. Env vars: `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL` (default `claude-sonnet-4-20250514`)

## Risks

- **Feature parity gap**: Anthropic's extended thinking uses a different mechanism than OpenAI's reasoning_effort. The thinking-level config needs provider-aware translation.
- **Wire format divergence**: Anthropic's Messages API has a different request/response shape (content blocks, tool_use vs function calls, different SSE events). The adapter is non-trivial.
- **Streaming SSE differences**: Anthropic uses `event: content_block_delta` with `delta.type` variants vs OpenAI's `choices[].delta` shape. Two separate SSE parsers needed.
- **Guard test impact**: `internal/core/guard_test.go` enforces stdlib-only imports in core. Provider adapters must stay in `internal/adapters/providers/` — no leakage.
- **Profile system coupling**: Adding `provider` to profile.yaml means profile resolution needs to carry provider alongside model, thinking, etc. The `modelLoaderAdapter` in runtime.go may need to become a broader adapter.
- **Reload interaction**: `runtime.Config.Client` factory is called on reload. If the provider name changes between reloads, the factory must produce a different concrete type. The current factory signature `func() (core.Provider, error)` already supports this — the closure captures the resolved provider name.

## Ready for Proposal

Yes — the exploration is complete. The recommendation is clear: provider registry + factory with Anthropic as the first new adapter. The orchestrator should tell the user:

> We currently have one hardcoded provider (OpenAI-compatible). Adding provider selection requires: (1) a provider registry that maps names to factory functions, (2) a `--provider` CLI flag and `provider:` profile.yaml field with layered resolution, (3) an Anthropic adapter for the Messages API. The biggest complexity is the Anthropic adapter itself — different wire format, auth, streaming, and thinking semantics. Estimated effort: Medium-High for the full change.

## Key Learnings

1. ADR-0003 explicitly deferred multi-provider support and names Anthropic as the first non-compatible provider to add.
2. The `runtime.Config.Client` factory signature already supports dynamic provider creation — it re-reads env on every reload.
3. The profile.yaml resolution system (3-layer: global → project → profile) needs a new `provider` field to support per-profile provider selection.
4. Thinking/extended-thinking has fundamentally different wire formats across providers — the adapter must handle provider-specific thinking semantics.
5. The hexagonal guard test (`core/guard_tests`) must not be violated — all provider adapters stay in `internal/adapters/providers/`.
