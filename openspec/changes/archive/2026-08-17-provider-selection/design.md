# Design: Provider Selection

## Technical Approach

Provider registry + factory function pattern (Approach 1 from exploration). A `map[string]ProviderFactory` maps provider names to constructors. Layered resolution picks the name; the registry produces the concrete client. OpenCode adapter reuses `openai.Client` with base URL override — zero new adapter code.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|-------------|-----------|
| Registry location | `internal/adapters/providers/registry.go` | cmd/kui or runtime | Keeps provider knowledge in adapters layer; cmd/kui stays thin; runtime gets a `ResolveProvider(name)` helper |
| OpenCode adapter | Reuse `openai.Client` with env override | New package | Same wire format (OpenAI-compatible); only base URL and API key env differ |
| `-p` short flag | Reassign from `--print` to `--provider` | Use `-pr` or no short flag | Specs require `-p`; `--print` has no short flag (acceptable since it's rarely used interactively) |
| Thinking capability | `SupportsThinking() bool` on concrete type | Capability interface system | Lightweight; avoids full capability registry for 2 providers; future providers add the method |
| Factory signature | `func(apiKey, baseURL string) (core.Provider, error)` | `func() (core.Provider, error)` | Decouples env var reading from construction; registry validates key at creation (fail-fast) |

## Data Flow

```
CLI args ──→ parseFlags() ──→ Options.Provider
                                    │
Profile.yaml ──→ Loader.Resolve() ──→ Profile.Provider
                                    │
Env var ─────→ KUI_PROVIDER ────────┘
                                    │
                         resolveProvider() → providerName
                                    │
                         registry[providerName] → factory
                                    │
                         factory(apiKey, baseURL) → core.Provider
                                    │
                         SupportsThinking() check → warn if mismatch
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/adapters/providers/registry.go` | Create | Provider registry: `map[string]ProviderFactory`, `Register()`, `Resolve()` |
| `internal/adapters/providers/opencode/` | Create | Thin adapter: reuses `openai.NewClient` with `OPENCODE_API_KEY` + `OPENCODE_BASE_URL` |
| `cmd/kui/flags.go` | Modify | Add `Provider string` to Options; reassign `-p` from `--print` to `--provider`; add `"provider"` to `stringFlags` |
| `cmd/kui/main.go` | Modify | Import registry; call `resolveProvider()` before `registry.Resolve()`; replace hardcoded `openai.NewClient()` |
| `internal/adapters/profile/loader.go` | Modify | Add `Provider string` to Config/Profile structs; carry through resolution |
| `internal/tui/run.go` | Modify | Accept resolved provider name; pass to registry factory |
| `internal/runtime/runtime.go` | Modify | `Config.Client` factory receives provider name from resolution chain |
| `internal/adapters/providers/openai/client.go` | Modify | Add `SupportsThinking() bool` method (returns true); accept baseURL param in constructor |

## Interfaces / Contracts

```go
// ProviderFactory creates a Provider from resolved configuration.
type ProviderFactory func(apiKey, baseURL string) (core.Provider, error)

// ProviderEntry registers a factory with its metadata.
type ProviderEntry struct {
    Factory          ProviderFactory
    RequiredEnvVar   string   // e.g. "OPENAI_API_KEY"
    BaseURLEnvVar    string   // e.g. "OPENAI_BASE_URL"
    DefaultBaseURL   string   // e.g. "https://api.openai.com/v1"
    SupportsThinking bool
}

// Registry maps provider names to entries.
type Registry struct {
    entries map[string]ProviderEntry
}

func (r *Registry) Register(name string, entry ProviderEntry)
func (r *Registry) Resolve(name string) (ProviderEntry, error)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Registry register/resolve; provider resolution chain; `-p` flag parsing | Table-driven tests with httptest for factory validation |
| Unit | Thinking degradation warning | Mock provider with `SupportsThinking()=false`; verify stderr warning |
| Integration | `--provider opencode` sends to correct base URL | httptest server with request assertion |
| Integration | Fail-fast on missing API key | Unset env var; verify error message names the variable |
| E2E | Provider switch preserves history | Not in scope for first slice |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No data migration required. Existing profiles without `provider:` field default to empty string, which resolves to `"openai"` via the resolution chain. The `-p` short flag reassignment is a breaking change for users who used `kui -p` (print mode) — documented in changelog; `--print` long form still works.

## Open Questions

- [ ] Should the registry live in `internal/adapters/providers/` or `internal/runtime/`? Design assumes adapters layer for hexagonal consistency.
- [ ] OpenCode base URL `https://opencode.ai/zen/go/v1` — confirm this is the correct endpoint.
