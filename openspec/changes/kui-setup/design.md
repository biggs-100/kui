# Design: kui setup

## Technical Approach

Add a credential storage adapter (`internal/credentials/store.go`) and a `kui setup` CLI subcommand (`cmd/kui/setup.go`). The credential store follows the existing `.kui/` persistence pattern from `internal/adapters/store/store.go`. The setup wizard uses `bufio.NewReader` for interactive input — no external dependencies. The provider resolution chain in `resolver.go` gains a credential-file lookup layer between env var and error.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|-------------|-----------|
| Package location | `internal/credentials/` (new package) | Extend `internal/adapters/store/` | Credential store is a distinct concern (API keys vs model memory). Keeps `store.go` focused on its current responsibility. |
| File format | JSON with `"providers"` map | YAML, env-file, per-provider files | JSON matches existing `models.json` pattern. Single file is simpler to manage. |
| Interactive input | `bufio.NewReader(os.Stdin)` | Bubbletea TUI, promptui library | Proposal explicitly scoped to CLI-only. Zero external deps. Matches project's stdlib preference. |
| Permission model | `0600` on Unix, best-effort on Windows | No permission control | REQ-CRED-3 mandates restrictive permissions. Windows doesn't support Unix perms; skip silently. |
| Validation | Format-only (prefix check) | Network ping to provider API | Network validation adds latency, requires API access, and may fail for valid keys with rate limits. Format check catches obvious mistakes. |

## Data Flow

### Setup wizard flow

    kui setup [--provider X]
        │
        ▼
    runSetup(root, args)
        │
        ├─── resolve provider (flag or interactive list)
        ├─── prompt API key (masked via terminal echo off)
        ├─── validate key (trim, non-empty, prefix check)
        ├─── CredentialStore.SetAPIKey(provider, key)
        │        │
        │        ▼
        │    .kui/credentials.json  (JSON, 0600)
        │
        └─── print success + next steps

### Credential resolution flow (modified CreateProvider)

    CreateProvider(reg, name)
        │
        ├─── os.Getenv(entry.RequiredEnvVar)  ──→ found? use it
        │
        ├─── CredentialStore.GetAPIKey(name)   ──→ found? use it
        │
        └─── error: "%s is not set: export %s ..."

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/credentials/store.go` | Create | `CredentialStore` — `GetAPIKey`, `SetAPIKey`, JSON read/write, 0600 perms |
| `internal/credentials/store_test.go` | Create | Unit tests for load/save/permissions/error cases |
| `cmd/kui/setup.go` | Create | `runSetup` — provider list, masked input, validation, save, success output |
| `cmd/kui/setup_test.go` | Create | Unit tests for flag parsing, validation, non-interactive mode |
| `internal/adapters/providers/resolver.go` | Modify | `CreateProvider` — add credential-file lookup after env var |
| `internal/adapters/providers/resolver_test.go` | Modify | Add tests for credential-file fallback and env-var precedence |
| `cmd/kui/main.go` | Modify | Add `setup` subcommand routing in `run()` |

## Interfaces / Contracts

```go
// internal/credentials/store.go

// CredentialStore manages provider API keys in .kui/credentials.json.
type CredentialStore struct {
    root string // project root (resolved from cwd or KUI_HOME)
}

func New(root string) *CredentialStore

// GetAPIKey returns the stored key for provider, or an error if not found.
func (cs *CredentialStore) GetAPIKey(provider string) (string, error)

// SetAPIKey persists the key for provider and writes the file to disk.
func (cs *CredentialStore) SetAPIKey(provider, key string) error
```

```go
// .kui/credentials.json format
{
  "providers": {
    "openai": { "api_key": "sk-..." },
    "opencode": { "api_key": "..." }
  }
}
```

```go
// cmd/kui/setup.go — entry point
func runSetup(root string, args []string) int
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `CredentialStore` load/save/permissions/errors | Table-driven tests with `t.TempDir()` |
| Unit | Key validation (empty, whitespace, prefix) | Pure function tests |
| Unit | `CreateProvider` credential-file fallback | Mock env + temp credentials file |
| Integration | `runSetup` non-interactive mode | `--provider` flag with piped stdin |
| Integration | Full wizard flow | teatest or stdin mock |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. New file `.kui/credentials.json` is created on first `kui setup`. Existing env-var-only users are unaffected — env vars remain the highest-priority resolution layer.

## Open Questions

- [ ] Should `kui setup` show current credentials status (which providers have keys saved)?
- [ ] Should `kui setup` support `--dry-run` to validate without saving?
