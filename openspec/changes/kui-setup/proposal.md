# Proposal: kui setup

## Intent

kui requires API keys via env vars (`OPENAI_API_KEY`, etc.), which is hostile for first-time users. A `kui setup` subcommand lets users interactively configure provider credentials from the CLI, persist them to disk, and run `kui tui` without manual env var management.

## Scope

### In Scope
- `kui setup` subcommand with interactive provider selection (list available providers from registry)
- API key input with basic format validation (non-empty, reasonable length)
- Persist credentials to `.kui/credentials.json` using the project's local config convention
- Layered resolution: env var > credentials file > profile config
- `kui setup --provider <name>` for non-interactive single-provider setup

### Out of Scope
- OAuth / web-based authentication flows
- TUI-mode setup (keep it CLI-only for v1)
- API key rotation or refresh logic
- Cloud/sync of credentials across devices
- Encrypted credential storage (defer to future hardening)

## Capabilities

### New Capabilities
- `credential-storage`: Credential file management — read/write/validate `.kui/credentials.json`, layered resolution with env vars

### Modified Capabilities
- `provider-selection`: Add credentials-file resolution layer between env var and profile config in the provider resolution chain

## Approach

1. Add `credential-storage` package: `Load()`, `Save()`, `Resolve(provider)` methods operating on `.kui/credentials.json`
2. Extend `provider-selection` to call `credential-storage.Resolve()` after env var lookup
3. Add `kui setup` subcommand to `cmd/kui/` using the hand-rolled flag parser pattern from `cli-flags`
4. Interactive prompts via `bufio.NewReader(os.Stdin)` — no external prompt library
5. Validate keys: non-empty, trim whitespace, basic format check per provider (e.g. `sk-` prefix for OpenAI)

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/credential/` | New | Credential store package |
| `internal/provider/` | Modified | Add credential resolution layer |
| `cmd/kui/` | Modified | Add `setup` subcommand routing |
| `.kui/credentials.json` | New | Persisted credential file |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Plaintext credential file on disk | Med | Document in README; file permissions 0600; defer encryption to v2 |
| Invalid keys not caught until runtime | Med | Validate format at setup time; warn on possible issues |
| Credential file format breaking change | Low | Version field in JSON; migration on read |

## Rollback Plan

1. Delete `.kui/credentials.json` (user data only)
2. Revert `internal/provider/` changes — env var resolution remains as fallback
3. Remove `cmd/kui/setup.go` and related tests

## Dependencies

- None external — uses stdlib `bufio`, `os`, `encoding/json`, `filepath`

## Success Criteria

- [ ] `kui setup` prompts for provider and API key, saves to `.kui/credentials.json`
- [ ] `kui setup --provider openai` accepts key non-interactively
- [ ] `kui tui` reads credentials from file when env vars are absent
- [ ] `kui setup` fails with clear error when given invalid/empty input
- [ ] Credentials file has restrictive permissions (0600 on Unix)
- [ ] All existing `go test ./...` pass after changes
