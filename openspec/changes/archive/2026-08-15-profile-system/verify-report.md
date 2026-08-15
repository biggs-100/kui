```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:26aaa8b5564610de09652ff5ba9024d528d827456716dd0b69f9f98d34402cd1
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 23/23
scenarios: 52/52
test_command: go test -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:e1f746aa1db07adcc63b28db66c88620a3008277afc2050f43244fb502b25e45
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: profile-system
**Version**: N/A (openspec change)
**Mode**: Strict TDD

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total | 30 |
| Tasks complete | 30 |
| Tasks incomplete | 0 |

### Build & Tests Execution
**Build**: ✅ Passed (`go build ./...`, exit 0, empty output)
**Tests**: ✅ 149 passed / 0 failed / 2 skipped (`go test -count=1 ./...`, exit 0; 2 skips are the pre-existing Windows symlink tests in `internal/adapters/tools`)
**Race**: ✅ `go test -count=1 -race ./internal/core/ ./internal/agent/ ./internal/adapters/...` — all ok
**Vet**: ✅ `go vet ./...` — exit 0
**Lint**: ✅ `golangci-lint run ./...` — 0 issues
**Gofmt**: ✅ 0 issues on changed files (LF-normalized; repo-wide CRLF-on-disk noise pre-existing)

**Coverage** (informational; no configured threshold):
| Package | Coverage |
|---------|----------|
| internal/core | 89.8% |
| internal/agent | 94.0% |
| adapters/permissions | 100.0% |
| adapters/profile | 90.0% |
| adapters/providers/openai | 86.4% |
| adapters/skills | 84.5% |
| adapters/store | 72.9% |
| adapters/tools | 86.5% |
| cmd/kui | 0% (exec-binary smoke; coverage does not cross the process boundary — expected, exercised via real binary) |

### Spec Compliance Matrix

| Requirement | Scenarios | Covering Test(s) | Result |
|-------------|-----------|------------------|--------|
| REQ-PROFILE-1 Profile Model | 3 (valid, malformed, missing SYSTEM.md) | `loader_test.go > TestResolveValidProfile`, `TestResolveMalformedYamlNamesFile`, `TestSystemPromptMissingBody`; `profile_manager_test.go > TestApplySwitchMissingSystemPrompt` | ✅ COMPLIANT |
| REQ-PROFILE-2 Layered Resolution | 2 (override precedence, empty profile layer) | `loader_test.go > TestResolveOverridePrecedence`, `TestResolveNearestWinsPerField`, `TestResolveEmptyLayersFallbackToGlobal` | ✅ COMPLIANT |
| REQ-PROFILE-3 Hot Switch Between Turns | 3 (switch while tool runs, history preserved, unknown profile) | `loop_test.go > TestRunSwitchAppliesBetweenTurns`, `TestRunUnknownProfileReturnsTypedError`; `loader_test.go > TestResolveUnknownProfile`; `profile_manager_test.go > TestApplySwitchUnknownProfile` | ✅ COMPLIANT |
| REQ-PROFILE-4 Model Memory | 2 (persist/restore, no saved model) | `store_test.go > TestStorePersistAndRestore`, `TestStoreGetNoSavedModelFallback`, `TestStoreSetPreservesOtherProfiles`, `TestStoreActiveRoundtrip` | ✅ COMPLIANT |
| REQ-PROFILE-5 Adapter Boundary | 2 (guard enforced, adapter implements port) | `core/guard_test.go > TestCoreImportsStdlibOnly`; `agent/guard_test.go > TestAgentImportsNoIOOrYaml`; adapter tests | ✅ COMPLIANT |
| REQ-PERM-1 Ruleset Evaluation | 4 (last wins, wildcard, empty, unregistered) | `ruleset_test.go > TestEvaluateLastRuleWins`, `TestEvaluateWildcardMatch`, `TestEvaluateEmptyRulesetAllows`, `TestEvaluateUnregisteredToolRuleIgnored`, `TestFlattenLayerOrder` | ✅ COMPLIANT |
| REQ-PERM-2 Ask Degrades to Deny | 1 (ask→deny) | `ruleset_test.go > TestEvaluateAskDegradesToDeny` | ✅ COMPLIANT |
| REQ-PERM-3 Tool Hiding from Payload | 2 (deny removes, allow keeps) | `client_test.go > TestChatPayloadDenyAllHidesEveryTool`, `TestChatPayloadKeepsAllowedTools`; `loop_test.go > TestRunFiltersDeniedToolsBeforeChat`; `ruleset_test.go > TestRulesetFilterDropsDeniedTools` | ✅ COMPLIANT |
| REQ-PERM-4 Execution Blocking | 1 (denied dispatch rejected, no side effect) | `loop_test.go > TestRunDeniedDispatchReturnsPermissionError` | ✅ COMPLIANT |
| REQ-SKILL-1 Layered Discovery | 2 (collision, aggregation) | `index_test.go > TestIndexCollisionNearestWins`, `TestIndexCollisionProjectOverGlobal`, `TestIndexLayeredAggregation` | ✅ COMPLIANT |
| REQ-SKILL-2 Index and Trigger Matching | 2 (match, no match) | `index_test.go > TestIndexTriggerMatch`, `TestIndexTriggerMatchCaseInsensitive`, `TestIndexMatchNoTrigger`, `TestIndexMatchEmptyIndex` | ✅ COMPLIANT |
| REQ-SKILL-3 On-Demand Content | 3 (index-only, load on invocation, missing body) | `index_test.go > TestIndexBuildsWithoutBodies`, `TestIndexLoadOnInvocation`, `TestIndexLoadMissingBody`; `agent_test.go > TestSystemMessagesIndexOnly`, `TestLoadSkillOnInvocation` | ✅ COMPLIANT |
| REQ-PCLI-1 List Profiles | 2 (active marker, no profiles) | `main_test.go > TestCLIProfileListMarksActive`, `TestCLIProfileListNoProfiles`; `loader_test.go > TestDiscoverListsProfileNames`, `TestDiscoverEmptyRoot` | ✅ COMPLIANT |
| REQ-PCLI-2 Switch Profile | 2 (known, unknown) | `main_test.go > TestCLIProfileSwitchKnown`, `TestCLIProfileSwitchUnknown` | ✅ COMPLIANT |
| REQ-PCLI-3 Per-Profile Model | 2 (persists, missing args) | `main_test.go > TestCLIProfileModelPersists`, `TestCLIProfileModelMissingArgument` | ✅ COMPLIANT |
| REQ-QUEUE-1 Steering Drain | 2 (drain all, one per turn) | `loop_test.go > TestRunSteeringDrainsAllBeforeNextRequest`, `TestRunSteeringDrainsOnePerTurn`; `queues_test.go > TestDrainAllMode`, `TestDrainOneAtATimeMode` | ✅ COMPLIANT |
| REQ-QUEUE-2 Follow-Up Drain | 2 (drains at stop, empty queue) | `loop_test.go > TestRunFollowUpDrainsAtStop`, `TestRunFollowUpEmptyStopsNormally`, `TestRunFollowUpWaitsForEmptySteering` | ✅ COMPLIANT |
| REQ-QUEUE-3 Termination Contract | 2 (budget with queues, tool failure skips injection) | `loop_test.go > TestRunBudgetCountsFollowUpContinuations`, `TestRunToolFailureSkipsQueuedSteering` | ✅ COMPLIANT |
| REQ-LOOP-1 Loop Execution (two-level, modified) | 4 (direct answer, multi-step tools, steering between turns, termination with empty queues) | `loop_test.go > TestRunDirectAnswerWithoutTools`, `TestRunMultiStepToolResolution`, `TestRunSteeringDrainsAllBeforeNextRequest`, `TestRunNilSafeWhenPortsUnset`, `TestRunFollowUpEmptyStopsNormally`; `agent_test.go > TestRunWiresSteeringAndReturnsAnswer` | ✅ COMPLIANT |
| REQ-LOOP-5 Profile Switch Between Turns | 2 (applies between turns, multi-switch last wins) | `loop_test.go > TestRunSwitchAppliesBetweenTurns`, `TestRunMultipleSwitchesLastWins`; `main_test.go > TestCLIProfileSwitchDashPrompt` | ✅ COMPLIANT |
| REQ-LOOP-6 Profile-Context Marker | 2 (marker on switch, no marker without) | `loop_test.go > TestRunSwitchAppliesBetweenTurns`, `TestRunNoMarkerWithoutSwitch`; `profile_manager_test.go > TestApplySwitchResolvesProfileAndReturnsMessages` | ✅ COMPLIANT |
| REQ-CLI-3 Profile Subcommands | 3 (list, switch unknown, model persists) | `main_test.go > TestCLIProfileListMarksActive`, `TestCLIProfileSwitchUnknown`, `TestCLIProfileModelPersists`, `TestCLIProfileModelUnknownProfile`, `TestCLIProfileModelMissingArgument`, `TestCLIProfileNoSubcommand` | ✅ COMPLIANT |
| REQ-CLI-4 Per-Profile Model Resolution | 2 (saved wins, fallback chain) | `main_test.go > TestCLIProfileModelResolutionSavedWins`, `TestCLIProfileModelFallbackToGlobal` | ✅ COMPLIANT |

**Compliance summary**: 52/52 scenarios compliant (all have a passing covering test)

### Correctness (Static Evidence)
| Requirement | Status | Notes |
|------------|--------|-------|
| Two-level loop (steering + follow-up) | ✅ Implemented | `core/loop.go` — additive nil-safe ports; termination contract preserved (D14) |
| Permission filter + dispatch guard | ✅ Implemented | Filter pre-Chat (REQ-PERM-3) + `Allow` guard → `PermissionError` (REQ-PERM-4) |
| Hot switch between turns | ✅ Implemented | `applySteering` on drain; last-switch-wins; history preserved; marker appended (REQ-LOOP-5/6) |
| Model memory `.kui/models.json` + `.kui/active` | ✅ Implemented | `adapters/store`, `KUI_HOME` hermetic override (D20) |
| Skills index without bodies; load on invocation | ✅ Implemented | `adapters/skills`, `skill.yaml` + `SKILL.md` (D21) |
| CLI `profile list\|switch\|model` + `--` form | ✅ Implemented | `cmd/kui/main.go`, exit 0/1/2, actionable stderr (D18) |
| REQ-CLI-4 resolution chain | ✅ Implemented | saved model → profile.yaml → OPENAI_MODEL → default; `SetModel` (D17) |

### Coherence (Design)
| Decision | Followed? | Notes |
|----------|-----------|-------|
| D14 additive two-level loop, `Chat` signature kept | ✅ Yes | `core/loop.go` nil-safe fields; 55 baseline behavior preserved |
| D15 loop-level filter + dispatch guard | ✅ Yes | Filter on `Tools.List()` pre-Chat; `Allow` → `PermissionError` |
| D16 `ApplySwitch` during steering drain, returns messages | ✅ Yes | `profile_manager.go`, marker + system prompt appended |
| D17 `openai.Client.SetModel` | ✅ Yes | Additive method; construction unchanged |
| D18 CLI dual-path switch + `KUI_HOME` | ✅ Yes | `.kui/active` persist + steering switch; hermetic tests |
| D19 mutex queue in `internal/agent` | ✅ Yes | `PendingMessageQueue`, `QueueMode`, race-clean |
| D20 `.kui/models.json` + `.kui/active` text store | ✅ Yes | `adapters/store` |
| D21 `skill.yaml` index + on-demand body | ✅ Yes | `adapters/skills` |
| PR 4/5 wiring (SystemMessages + SYSTEM.md via steering) | ⚠️ Deviation | See WARNING-1/WARNING-2 — documented, tested, but deviates from spec wording and design data flow |

### TDD Compliance
| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | ✅ | apply-progress found in Engram with "TDD Cycle Evidence" table |
| All tasks have tests | ✅ | 30/30 tasks map to test files; RED "✅ Written" |
| RED confirmed (tests exist) | ✅ | All referenced test files verified to exist |
| GREEN confirmed (tests pass) | ✅ | 149/149 executed tests pass (0 fail) |
| Triangulation adequate | ✅ | Multiple cases per behavior (e.g., 4 ruleset-eval cases, 3 skill scenarios) |
| Safety Net for modified files | ⚠️ | Retained apply-progress carries per-task RED/GREEN rows only for the final batch (PR 5); phases 1–4 marked complete without per-task rows in the retained revision (verified independently: files exist + pass) |

**TDD Compliance**: 5/6 checks passed (1 ⚠️ evidence-completeness)

### Test Layer Distribution
| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | ~146 | 17 test files | stdlib testing |
| Integration | 13 exec-binary smoke + httptest payload | cmd/kui/main_test.go, client_test.go | os/exec, net/http/httptest |
| E2E | 0 | — | N/A (CLI binary, no browser) |
| **Total** | **149 passed + 2 skipped** | **9 packages** | |

### Assertion Quality
| File | Line | Assertion | Issue | Severity |
|------|------|-----------|-------|----------|
| — | — | — | None found | — |

**Assertion quality**: ✅ All assertions verify real behavior (value + role + payload assertions throughout; no tautologies, ghost loops, or type-only assertions; fakes are hand-rolled, not mock-heavy)

### Quality Metrics
**Linter**: ✅ No errors (0 issues)
**Type Checker**: ✅ No errors (`go vet ./...` clean)
**Coverage**: informational — lowest changed-file package is `adapters/store` at 72.9%; others ≥ 84%

### Issues Found

**CRITICAL**: None

**WARNING**:
1. **Skills index rides as a USER-role steering message, not a system-prompt message (PR 4/5 nuance).** `agent.SystemMessages()` builds the skills index as a `RoleSystem` message, but `cmd/kui/main.go` extracts only its `.Content` and enqueues it as a steering `PendingMessage{Content}`, which `core/loop.go:applySteering` injects as `RoleUser`. REQ-SKILL-3 says the skills index belongs in the *system prompt* ("The system prompt MUST contain only the skills index"). The observable scenario assertions (index lists names/triggers, no body text) still pass — `TestSystemMessagesIndexOnly` asserts RoleSystem at the wrapper level — so no scenario is broken, but the wire role deviates from the requirement's wording. Not fixed; reported per instructions.
2. **Profile system context arrives only from request 2, not request 1 (PR 4/5 nuance).** At session start `runPrompt` calls `manager.ApplySwitch(ctx, activeName)` and discards its returned messages, then enqueues a `SwitchProfile` + skills index into the steering queue. The first provider request therefore carries neither the profile SYSTEM.md nor the skills index; both are injected after the first turn completes (second request). This is documented in code ("PR 4 note") and asserted by `TestCLIProfileSystemPromptInjected`/`TestCLIProfileSwitchDashPrompt` (they check `bodies[1]`). It deviates from the design data-flow diagram (SYSTEM.md + skills index → SystemMessages seeded before the first request) and means the profile's system prompt never shapes the first turn. No scenario requires first-request context, so no scenario breaks — flagged as a design-coherence deviation. Not fixed; reported per instructions.
3. **apply-progress TDD evidence completeness.** The retained apply-progress (Engram, topic `sdd/profile-system/apply-progress`) carries the per-task "TDD Cycle Evidence" rows only for the final batch (PR 5, tasks 5.1–6.2); phases 1–4 are marked complete with commit refs but no per-task RED/GREEN rows in the retained revision (they were in prior revisions of the upserted observation). Independent verification confirms all test files exist and pass, so this is an evidence-retention gap, not a protocol failure.

**SUGGESTION**:
1. `cmd/kui` shows 0% under `go test -cover` because the exec-binary smoke pattern runs coverage in a child process that is not attributed to the package. Consider an integration coverage harness (e.g., `-coverpkg` with process coverage or a golden-run recording) if per-command coverage is ever desired — informational only, not a failure.
2. Consider seeding the active profile's SYSTEM.md and the skills index into the first provider request at session start (instead of the steering queue's request-2 delivery) to match the design data flow and give the first turn full profile context.

### Verdict
**PASS WITH WARNINGS**
All 30/30 tasks complete; all 23 requirements and 52 scenarios have passing covering tests; build, vet, lint, gofmt, and race are clean; 149 tests green with 0 failures. Three WARNING-level findings (all non-blocking, none breaking a passing scenario): two documenting the known PR 4/5 wiring deviations (skills index role + request-2 system context delivery), and one on apply-progress TDD-evidence retention for earlier phases.
