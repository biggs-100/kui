```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:a652db0435c197fb04cc1abb28ced6e7ee8d74ccc2d41005a636d287bc04e27d
verdict: pass
blockers: 0
critical_findings: 0
requirements: 9/9
scenarios: 19/19
test_command: go test -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:a652db0435c197fb04cc1abb28ced6e7ee8d74ccc2d41005a636d287bc04e27d
build_command: go build ./cmd/kui/
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: provider-selection
**Version**: N/A
**Mode**: Strict TDD

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total | 38 |
| Tasks complete | 38 |
| Tasks incomplete | 0 |

### Build & Tests Execution
**Build**: ✅ Passed
```text
go build ./cmd/kui/
(exit 0, no output)
```

**Tests**: ✅ 17 packages passed / ❌ 0 failed / ⚠️ 0 skipped
```text
ok  github.com/biggs-100/kui/cmd/kui                                  13.772s
ok  github.com/biggs-100/kui/internal/adapters/extensions               0.995s
ok  github.com/biggs-100/kui/internal/adapters/permissions              0.946s
ok  github.com/biggs-100/kui/internal/adapters/profile                  1.468s
ok  github.com/biggs-100/kui/internal/adapters/providers                2.740s
ok  github.com/biggs-100/kui/internal/adapters/providers/openai         3.289s
ok  github.com/biggs-100/kui/internal/adapters/providers/opencode       2.112s
ok  github.com/biggs-100/kui/internal/adapters/skills                   4.121s
ok  github.com/biggs-100/kui/internal/adapters/store                    1.288s
ok  github.com/biggs-100/kui/internal/adapters/tools                    4.299s
ok  github.com/biggs-100/kui/internal/agent                             4.070s
ok  github.com/biggs-100/kui/internal/core                              1.484s
ok  github.com/biggs-100/kui/internal/extensions/example                1.031s
ok  github.com/biggs-100/kui/internal/mcp                               1.515s
ok  github.com/biggs-100/kui/internal/runtime                           1.603s
ok  github.com/biggs-100/kui/internal/tui                               3.350s
ok  github.com/biggs-100/kui/internal/tui/views                         0.433s
```

**Vet**: ✅ Passed (no output)

### Spec Compliance Matrix

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| REQ-SEL-1 | Known provider lookup | `providers/registry_test.go > TestRegistryKnownProviderLookup` | ✅ COMPLIANT |
| REQ-SEL-1 | Unknown provider lookup | `providers/registry_test.go > TestRegistryUnknownProviderLookup` | ✅ COMPLIANT |
| REQ-SEL-2 | Flag overrides profile | `providers/resolver_test.go > TestResolveProvider_FlagTakesPrecedence` | ✅ COMPLIANT |
| REQ-SEL-2 | Default fallback | `providers/resolver_test.go > TestResolveProvider_DefaultOpenAI` | ✅ COMPLIANT |
| REQ-SEL-3 | Key present | `providers/resolver_test.go > TestCreateProvider_UsesDefaultBaseURL` | ✅ COMPLIANT |
| REQ-SEL-3 | Key missing | `providers/resolver_test.go > TestCreateProvider_MissingAPIKey` | ✅ COMPLIANT |
| REQ-SEL-4 | Provider supports thinking | `providers/integration_test.go > TestRegistryOpenAIThinkingSupport` | ✅ COMPLIANT |
| REQ-SEL-4 | Provider lacks thinking | `providers/integration_test.go > TestRegistryOpenCodeNoThinkingSupport` | ✅ COMPLIANT |
| REQ-THINK-13 | Provider supports thinking | `providers/integration_test.go > TestThinkingDegradationNoWarningWhenSupported` | ✅ COMPLIANT |
| REQ-THINK-13 | Provider lacks thinking | `providers/integration_test.go > TestThinkingDegradationWarningEmitted` | ✅ COMPLIANT |
| REQ-PROV-3 | Custom base URL | `opencode/client_test.go > TestOpenCodeBaseURLOverride` | ✅ COMPLIANT |
| REQ-PROV-3 | Default base URL | `providers/resolver_test.go > TestCreateProvider_UsesDefaultBaseURL` + `opencode/client_test.go > TestOpenCodeDefaultBaseURLFallback` | ✅ COMPLIANT |
| REQ-PROV-3 | Per-provider base URL env var | `providers/resolver_test.go > TestCreateProvider_UsesEnvBaseURL` | ✅ COMPLIANT |
| REQ-PROFILE-1 | Valid profile with provider | `profile/loader_test.go > TestResolveProviderFromProfile` | ✅ COMPLIANT |
| REQ-PROFILE-1 | Valid profile without provider | `profile/loader_test.go > TestResolveProviderEmpty` | ✅ COMPLIANT |
| REQ-PROFILE-1 | Malformed yaml | `profile/loader_test.go > TestResolveMalformedYamlNamesFile` | ✅ COMPLIANT |
| REQ-CLI-10 | All fields default | `cmd/kui/flags_test.go > TestOptionsZeroValues` | ✅ COMPLIANT |
| REQ-CLI-10 | Provider flag set | `cmd/kui/flags_test.go > TestParseFlagsProviderLong` | ✅ COMPLIANT |
| REQ-CLI-10 | Provider short flag | `cmd/kui/flags_test.go > TestParseFlagsProviderShort` | ✅ COMPLIANT |

**Compliance summary**: 19/19 scenarios compliant

### TDD Compliance
| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | ✅ | Found in engram apply-progress artifact |
| All tasks have tests | ✅ | 38/38 tasks complete; RED/GREEN columns filled for core tasks |
| RED confirmed (tests exist) | ✅ | 8/8 RED tasks have test files verified in codebase |
| GREEN confirmed (tests pass) | ✅ | 8/8 GREEN tasks pass on execution |
| Triangulation adequate | ✅ | Phase 6 tests added: 4 resolver tests, 3 thinking degradation tests, 1 opencode fallback |
| Safety Net for modified files | ✅ | 4/4 modified files have safety net (resolver_test.go, integration_test.go, opencode/client_test.go, flags_test.go) |

**TDD Compliance**: 6/6 checks passed

### Test Layer Distribution
| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | ~75 | 5 | go test |
| Integration | 7 | 1 | go test + httptest |
| E2E | 0 | 0 | — |
| **Total** | **~82** | **6** | |

### Correctness (Static Evidence)
| Requirement | Status | Notes |
|------------|--------|-------|
| REQ-SEL-1 | ✅ Implemented | Registry maps names → ProviderEntry with factory, env vars, capabilities. NewDefaultRegistry registers openai + opencode. |
| REQ-SEL-2 | ✅ Implemented | `ResolveProvider()` in resolver.go implements flag → profile → env → default chain. Tested via 4 unit tests. |
| REQ-SEL-3 | ✅ Implemented | `CreateProvider()` in resolver.go reads RequiredEnvVar, fails if empty. Factory constructs client from resolved values. Tested via 5 unit tests. |
| REQ-SEL-4 | ✅ Implemented | `WarnThinkingDegradation()` in resolver.go checks SupportsThinking() via type assertion. Tested via 3 integration tests with mock provider. |
| REQ-PROV-3 | ✅ Implemented | Per-provider BaseURLEnvVar in ProviderEntry; factory receives resolved baseURL. Default fallback tested via resolver_test.go and opencode/client_test.go. |
| REQ-PROFILE-1 | ✅ Implemented | Provider string field in Config and Profile structs; nearest-wins merge in resolve(). |
| REQ-CLI-10 | ✅ Implemented | Provider string field in Options; -p reassigned from print to provider; stringFlags includes "provider". |
| REQ-THINK-13 | ✅ Implemented | openai.Client.SupportsThinking() returns true; opencode has no such method (metadata only). Warning tested with mock provider. |

### Coherence (Design)
| Decision | Followed? | Notes |
|----------|-----------|-------|
| Registry location in `internal/adapters/providers/` | ✅ Yes | registry.go placed correctly |
| OpenCode reuses openai.Client with env override | ✅ Yes | opencode.NewClient delegates to openai.NewClientWithConfig |
| `-p` short flag reassigned to --provider | ✅ Yes | shortMap["p"] = "provider"; --print has no short flag |
| Thinking capability via SupportsThinking() bool | ✅ Yes | Type assertion on concrete client; avoids interface system |
| Factory signature `func(apiKey, baseURL string) (core.Provider, error)` | ✅ Yes | Matches design exactly |
| Layered resolution chain in ResolveProvider() | ✅ Yes | flag → profile → env → "openai" default; extracted to resolver.go |
| Per-provider env var reading in CreateProvider() | ✅ Yes | Entry.RequiredEnvVar and Entry.BaseURLEnvVar used; extracted to resolver.go |
| Testability via io.Writer for warning output | ✅ Yes | WarnThinkingDegradation accepts io.Writer; tests use bytes.Buffer |

### Assertion Quality
✅ All assertions verify real behavior — no tautologies, trivial checks, or ghost loops found. Tests assert specific error messages contain provider names, field values match expected strings, and type assertions succeed.

### Issues Found

**CRITICAL**: None

**WARNING**: None

**SUGGESTION**: None

### Verdict
**PASS**

All 38 tasks are complete. All 9 requirements are implemented and verified with runtime test evidence. 19/19 spec scenarios have passing tests — all previously UNTESTED scenarios now have dedicated unit tests:

- **REQ-SEL-2** (previously UNTESTED): `TestResolveProvider_FlagTakesPrecedence`, `TestResolveProvider_ProfileTakesPrecedenceOverEnv`, `TestResolveProvider_EnvTakesPrecedenceOverDefault`, `TestResolveProvider_DefaultOpenAI` — 4 tests cover every branch of the resolution chain.
- **REQ-PROV-3** (previously UNTESTED): `TestCreateProvider_UsesDefaultBaseURL`, `TestCreateProvider_UsesEnvBaseURL`, `TestOpenCodeDefaultBaseURLFallback` — 3 tests cover default fallback, env override, and opencode-specific default.
- **Thinking degradation** (previously UNTESTED): `TestThinkingDegradationWarningEmitted`, `TestThinkingDegradationNoWarningWhenOff`, `TestThinkingDegradationNoWarningWhenSupported` — 3 tests with mock provider prove warning logic via `io.Writer` injection.

The `resolveProvider()` and `createProvider()` functions in `cmd/kui/main.go` now delegate to `providers.ResolveProvider()` and `providers.CreateProvider()` — pure functions with full unit test coverage. No linter issues were introduced by this change.
