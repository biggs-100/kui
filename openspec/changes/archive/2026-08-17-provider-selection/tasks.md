# Tasks: Provider Selection

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 280–380 |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | auto-chain |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Registry + OpenCode adapter + flags + profile loader + integration | Single PR | `go test ./internal/adapters/providers/... ./cmd/kui/... ./internal/adapters/profile/...` | `kui --provider opencode "hello"` | Entire change is atomic; revert single commit |

## Phase 1: Foundation — Registry + OpenCode Adapter

- [x] 1.1 Create `internal/adapters/providers/registry.go`: `ProviderFactory` type, `ProviderEntry` struct (Factory, RequiredEnvVar, BaseURLEnvVar, DefaultBaseURL, SupportsThinking), `Registry` with `Register()` and `Resolve(name)` methods. `Resolve` returns error naming unknown provider.
- [x] 1.2 RED: Create `internal/adapters/providers/registry_test.go` — table-driven tests: known provider lookup returns entry, unknown provider returns actionable error naming it.
- [x] 1.3 GREEN: Implement registry to pass tests from 1.2.
- [x] 1.4 Create `internal/adapters/providers/opencode/client.go`: thin adapter reusing `openai.NewClient` with env vars `OPENCODE_API_KEY` (required) and `OPENCODE_BASE_URL` (default `https://opencode.ai/zen/go/v1`). Factory signature: `func(apiKey, baseURL string) (core.Provider, error)`.
- [x] 1.5 RED: Create `internal/adapters/providers/opencode/client_test.go` — test: missing `OPENCODE_API_KEY` fails with error naming variable; httptest server verifies base URL override works.
- [x] 1.6 GREEN: Implement OpenCode adapter to pass tests from 1.5.
- [x] 1.7 Register `"openai"` and `"opencode"` in a package-level `init()` or a `NewDefaultRegistry()` function in `registry.go`.

## Phase 2: CLI Flags + Profile Loader

- [x] 2.1 Modify `cmd/kui/flags.go`: add `Provider string` to `Options` struct; add `"provider": true` to `stringFlags`; reassign `"p"` short flag from `""` (print bool) to `"provider"` in `shortMap`.
- [x] 2.2 Add `case "provider": opts.Provider = value` to `setStringOption()`.
- [x] 2.3 RED: Add flag tests in existing test file or new `cmd/kui/flags_test.go` — `--provider openai` sets `Options.Provider`; `-p opencode` sets `Options.Provider`; default is empty string.
- [x] 2.4 GREEN: Verify flag parsing passes tests from 2.3.
- [x] 2.5 Modify `internal/adapters/profile/loader.go`: add `Provider string` field to `Config` and `Profile` structs; carry through `resolve()` merge (same nearest-wins pattern as Model).
- [x] 2.6 RED: Add profile loader tests — profile with `provider: opencode` parses correctly; profile without provider field has empty string.
- [x] 2.7 GREEN: Verify loader passes tests from 2.6.

## Phase 3: Integration — Wire Registry into CLI/TUI/Runtime

- [x] 3.1 Modify `internal/runtime/runtime.go`: add `ProviderName string` to `Config` struct; in `Build()`, pass provider name to registry resolve before calling `cfg.Client()`.
- [x] 3.2 Modify `cmd/kui/main.go`: import registry; add `resolveProvider()` function implementing chain: `opts.Provider` → profile provider → `KUI_PROVIDER` env → default `"openai"`; replace `openai.NewClient()` with registry factory call.
- [x] 3.3 Modify `cmd/kui/main.go` `runTUI()`: pass resolved provider name in `tui.Wiring.Client` closure via registry.
- [x] 3.4 Modify `internal/tui/run.go`: accept provider name in `Wiring` or resolve via registry inside the `Client` factory closure (minimal change — keep `Client func()` signature).
- [x] 3.5 Add `SupportsThinking() bool` method to `internal/adapters/providers/openai/client.go` (returns `true`).
- [x] 3.6 Implement thinking degradation: after provider creation in `runPrompt()` and `runTUI()`, check `SupportsThinking()` — if thinking is configured but unsupported, emit `fmt.Fprintf(os.Stderr, "kui: WARNING: provider %q does not support thinking; continuing without it\n", name)`.

## Phase 4: Testing — End-to-End Verification

- [x] 4.1 RED: Add integration test — httptest server as OpenCode endpoint; create provider via registry with `OPENCODE_API_KEY` set; verify requests hit correct base URL path `/v1/chat/completions`.
- [x] 4.2 GREEN: Verify integration test passes.
- [x] 4.3 Verify all existing tests still pass: `go test ./...`.
- [x] 4.4 Verify build: `go build ./cmd/kui/`.

## Phase 5: Documentation + Cleanup

- [x] 5.1 Update `cmd/kui/main.go` usage string: add `--provider, -p <provider>` flag description; add `KUI_PROVIDER` to environment section.
- [x] 5.2 Remove any dead code or TODO comments from flag reassignment.
- [x] 5.3 Final `go vet ./...` and `golangci-lint run ./...` clean.

## Phase 6: Verify Warning Fixes — Testability Extraction

- [x] 6.1 Extract `resolveProvider()` and `createProvider()` to `internal/adapters/providers/resolver.go` as exported pure functions (`ResolveProvider`, `CreateProvider`). `CreateProvider` accepts `*Registry` parameter for testability.
- [x] 6.2 RED: Add unit tests for `ResolveProvider` — flag precedence, profile precedence, env precedence, default "openai" (4 tests in `resolver_test.go`).
- [x] 6.3 GREEN: Implement resolver to pass tests from 6.2.
- [x] 6.4 RED: Add `TestOpenCodeDefaultBaseURLFallback` in `opencode/client_test.go` — sets `OPENCODE_API_KEY` without `OPENCODE_BASE_URL`, verifies default URL via reflection.
- [x] 6.5 GREEN: Verify opencode fallback test passes.
- [x] 6.6 Extract `WarnThinkingDegradation()` to `resolver.go` with `io.Writer` parameter for testability. Update `cmd/kui/main.go` to call extracted function.
- [x] 6.7 RED: Add 3 thinking degradation tests in `integration_test.go` — mock provider with `SupportsThinking()=false`, warning emitted when thinking enabled, no warning when "off", no warning when supported.
- [x] 6.8 GREEN: All tests pass. Verify full suite: `go test ./...` (17 packages).
