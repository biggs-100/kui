# Proposal: Provider Selection

## Intent

kui hardcodes a single OpenAI-compatible provider. Users cannot switch providers without editing env vars. This change adds a `--provider` flag and `provider:` profile.yaml field so users select between providers per-session or per-profile, with graceful thinking degradation and fail-fast API key validation.

## Scope

### In Scope
- Provider registry mapping names → factory functions (`cmd/kui/` or `internal/runtime/`)
- `--provider` / `-p` CLI flag with layered resolution: flag → profile → env → default `openai`
- `provider:` field in `profile.yaml` (layered resolution alongside model)
- OpenCode provider adapter using existing `openai.Client` with base URL override (`https://opencode.ai/zen/go/v1`)
- Fail-fast API key validation at provider creation (before first prompt)
- Thinking degradation: warn when thinking is configured but provider doesn't support it; continue normally
- Session preservation on provider switch (history kept; compact on context window mismatch)
- Independent provider/model axes — provider does NOT change the default model

### Out of Scope
- Anthropic adapter (deferred to next change)
- Capability interface system (future; providers get simple `Capabilities() []string` hints for now)
- Model catalog per provider (model resolution stays user-driven)

## Capabilities

### New Capabilities
- `provider-selection`: Provider registry, `--provider` flag, `provider:` profile.yaml field, layered resolution chain, fail-fast API key validation, thinking degradation warnings

### Modified Capabilities
- `provider-openai-compatible`: Base URL becomes configurable per-provider (currently reads `OPENAI_BASE_URL` only)
- `profile-runtime`: Add `Provider string` field to profile.yaml; resolution carries provider alongside model
- `cli-flags`: Add `Provider string` to Options struct; `--provider` / `-p` flag support
- `thinking-provider`: Add thinking-capability check — warn and continue when provider lacks thinking support

## Approach

Provider registry + factory function (Approach 1 from exploration). Each provider is a package under `internal/adapters/providers/` with a `NewClient` factory. The registry maps `"openai"` and `"opencode"` to their factories. Resolution chain: `--provider` flag → `profile.yaml provider` → `KUI_PROVIDER` env → default `"openai"`.

OpenCode adapter reuses `openai.Client` with base URL `https://opencode.ai/zen/go/v1` and `OPENCODE_API_KEY`. Zero new adapter code for first slice.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/kui/flags.go` | Modified | Add `Provider` to Options; parse `--provider`/`-p` |
| `cmd/kui/main.go` | Modified | Dynamic provider creation via registry |
| `internal/tui/run.go` | Modified | Use registry instead of hardcoded `openai.NewClient()` |
| `internal/adapters/profile/loader.go` | Modified | Parse `provider:` from profile.yaml |
| `internal/runtime/runtime.go` | Modified | Config.Client factory receives resolved provider name |
| `internal/adapters/providers/opencode/` | New | OpenCode adapter (thin wrapper over openai.Client) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| OpenCode base URL changes | Low | Env var `OPENCODE_BASE_URL` override; fail fast on 404 |
| Thinking config silently ignored | Medium | Warn at startup and on provider switch; user sees message |
| Provider/model confusion | Low | Docs + error message: "provider and model are independent" |

## Rollback Plan

Revert the commit. Registry reverts to hardcoded `openai.NewClient()`. `--provider` flag becomes unknown-flag error (acceptable since it's new). Profile `provider:` field ignored by old loader. No data migration needed.

## Dependencies

- None external; OpenCode Zen endpoint must be reachable

## Success Criteria

- [ ] `kui --provider opencode` sends requests to OpenCode Zen endpoint
- [ ] `kui --provider openai` (default) works unchanged
- [ ] Missing API key produces actionable error before first prompt
- [ ] Thinking configured on OpenCode provider logs a warning, does not error
- [ ] Provider switch preserves conversation history
