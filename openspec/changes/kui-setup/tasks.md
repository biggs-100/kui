# Tasks: kui setup

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 450–550 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Credential Store) → PR 2 (Setup Wizard) → PR 3 (Integration) |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Credential store + tests | PR 1 | `go test ./internal/credentials/...` | N/A — pure library | `internal/credentials/` (new pkg, revert-safe) |
| 2 | Setup wizard + tests | PR 2 | `go test ./cmd/kui/ -run TestSetup` | `echo "key" \| kui setup --provider openai` | `cmd/kui/setup.go` + `cmd/kui/setup_test.go` |
| 3 | Resolver integration + main.go wiring | PR 3 | `go test ./...` | `kui tui` with credentials file present | `internal/adapters/providers/resolver.go`, `cmd/kui/main.go` changes |

## Phase 1: Credential Store (Foundation)

- [x] 1.1 **RED**: Create `internal/credentials/store_test.go` — tests for `New`, `GetAPIKey`, `SetAPIKey`, empty-file load, malformed-JSON error, 0600 permissions (REQ-CRED-1..5)
- [x] 1.2 **GREEN**: Create `internal/credentials/store.go` — `CredentialStore` struct, `New(root)`, `GetAPIKey(provider)`, `SetAPIKey(provider, key)`, JSON read/write to `.kui/credentials.json`, `os.MkdirAll`, `0600` perms on Unix
- [x] 1.3 **REFACTOR**: Extract `credFile` helper, ensure `readCreds`/`writeCreds` follow `store.go` patterns (readModels/writeModels), verify `go vet ./...` clean

## Phase 2: Setup Wizard

- [x] 2.1 **RED**: Create `cmd/kui/setup_test.go` — test flag parsing (`--provider`), empty-input rejection, masked-input validation, `ValidateKey` format checks (REQ-CRED-6, REQ-CRED-7)
- [x] 2.2 **GREEN**: Create `cmd/kui/setup.go` — `runSetup(root, args []string) int` entry point, parse `--provider` flag, list providers from registry, prompt API key via `bufio.NewReader`, masked echo off, `ValidateKey(provider, key)` function
- [x] 2.3 Add `ValidateKey(provider, key string) error` — non-empty, trim whitespace, prefix check per provider (`sk-` for openai), return descriptive error on failure (REQ-CRED-7)
- [x] 2.4 Wire save: `CredentialStore.SetAPIKey(provider, key)`, print success + next-steps message (REQ-CRED-8)
- [x] 2.5 Update `cmd/kui/main.go` — add `setup` case in `run()` function, before `profile` case, dispatching to `runSetup(root, args[1:])`
- [x] 2.6 Update `usage` string in `cmd/kui/main.go` — add `kui setup [--provider <name>]` subcommand line

## Phase 3: Integration (Resolver Fallback)

- [x] 3.1 **RED**: Update `internal/adapters/providers/resolver_test.go` — add tests: env var precedence over file, key found in file when env unset, key missing everywhere errors (REQ-SEL-3 scenarios)
- [x] 3.2 **GREEN**: Update `internal/adapters/providers/resolver.go` `CreateProvider` — after `os.Getenv` returns empty, call `credentials.New(root).GetAPIKey(name)`, use result if found; propagate error if missing everywhere
- [x] 3.3 Pass project root through to `CreateProvider` (add `root string` parameter or accept it via struct/option) so credential store resolves `.kui/credentials.json` relative to project root

## Phase 4: Cleanup / Verification

- [x] 4.1 Run `go vet ./...` and `golangci-lint run ./...` — ensure clean
- [x] 4.2 Run `go test ./...` — all existing + new tests pass
- [x] 4.3 Verify `go build ./cmd/kui` compiles successfully
