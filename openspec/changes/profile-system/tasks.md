# Tasks: Profile System — First-Class Profiles with Hot Switching

## Review Workload Forecast

Estimated changed lines: 1200–1500. Delivery: auto-chain.

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

1. **PR 1** ports + loop restructure — `go test ./internal/core/`; harness `go test ./...`; rollback core queue/loop/ports+tests
2. **PR 2** permissions + tool hiding — `go test ./internal/adapters/permissions/ ./internal/core/`; harness httptest payload (REQ-PERM-3); rollback ruleset, loop guard
3. **PR 3** loader/resolver + manager + switch — `go test ./internal/adapters/profile/ ./internal/agent/`; harness between-turn switch; rollback adapters/profile, manager, loop switch
4. **PR 4** skills + store + agent wrapper — `go test ./internal/adapters/...`; harness t.TempDir() + .kui/; rollback adapters/skills, store, agent/
5. **PR 5** SetModel + CLI + smoke — `go test ./...`; harness exec list/switch/model; rollback SetModel, main.go

## Phase 1: Core Ports + Two-Level Loop

- [x] 1.1 RED `loop_test.go`: drain all/one-per-turn, follow-up at stop, budget+queues, failure skips injection (REQ-QUEUE-1..3)
- [x] 1.2 Create `core/queue.go`: `PendingMessage`, `PendingQueue`, `QueueMode`
- [x] 1.3 Create `core/profile.go` + `permissions.go`: `ProfileManager`, `ModelMemory`, `PermissionEvaluator`
- [x] 1.4 Extend `core/errors.go`: `UnknownProfileError`, `PermissionError`, `ProfileActivationError`, `SkillLoadError`, `StoreError`
- [x] 1.5 Modify `core/loop.go`: nil-safe `Steering`/`FollowUp`/`Profiles`/`Permissions`; two-level `Run`
- [x] 1.6 Add `RoleSystem` to `core/provider.go`

## Phase 2: Permissions

- [x] 2.1 RED `loop_test.go`: filtered tools slice to Chat; denied dispatch → `PermissionError` (REQ-PERM-4)
- [x] 2.2 RED payload: httptest body excludes denied, keeps allowed (REQ-PERM-3)
- [x] 2.3 yaml.v3 resolved: NOT added in this slice — rules are constructed programmatically; `Flatten` consumes `[]Rule`. The yaml loader lands with PR 3 (`adapters/profile/loader.go`), keeping core stdlib-only
- [x] 2.4 Create `internal/adapters/permissions/ruleset.go`: `Flatten`, `Evaluate` (findLast, `path.Match`), Ask→Deny, `Filter`
- [x] 2.5 Create `ruleset_test.go`: last-wins, wildcard, empty, unregistered, ask→deny
- [x] 2.6 Modify `loop.go`: filter `Tools.List()` pre-Chat; `Allow` guard

## Phase 3: Profile Runtime + Hot Switch

- [ ] 3.1 RED `loop_test.go`: switch between turns, unknown profile, last-wins, marker (REQ-PROFILE-3, REQ-LOOP-5/6)
- [ ] 3.2 Create `internal/adapters/profile/loader.go`: yaml parse, layered discovery, pure resolver (REQ-PROFILE-1/2)
- [ ] 3.3 Create `loader_test.go`: valid, malformed names file, missing SYSTEM.md, precedence
- [ ] 3.4 Create `internal/agent/profile_manager.go`: `ApplySwitch` — resolve, SYSTEM.md, rebuild evaluator+registry, `SetModel`, marker
- [ ] 3.5 Modify `loop.go`: apply `Profiles.ApplySwitch` on steering drain

## Phase 4: Skills + Store + Agent Wrapper

- [ ] 4.1 Create `internal/adapters/skills/index.go`: layered discovery, `skill.yaml` index, trigger `Match`, body `Load` (REQ-SKILL-1..3)
- [ ] 4.2 Create `index_test.go`: collision, aggregation, index w/o bodies, load-on-invocation, missing body
- [ ] 4.3 Create `internal/adapters/store/store.go`: `.kui/models.json`, `.kui/active`, `KUI_HOME` (REQ-PROFILE-4)
- [ ] 4.4 Create `store_test.go`: persist/restore, no-saved-model fallback
- [ ] 4.5 Create `agent/queues.go`: mutex `PendingMessageQueue`, `Enqueue`/`Drain` per mode
- [ ] 4.6 Create `agent/agent.go`: active profile, queues, SystemMessages (skills index), `Run`, `LoadSkill`
- [ ] 4.7 Guard test `agent/`: no yaml/filesystem imports

## Phase 5: SetModel + CLI

- [ ] 5.1 Add `SetModel` to `adapters/providers/openai/client.go` (D17)
- [ ] 5.2 Extend `client_test.go`: `SetModel` changes request model
- [ ] 5.3 Modify `cmd/kui/main.go`: `list|switch|model` subcommands, resolution chain (REQ-CLI-4), `-- <prompt>`, usage
- [ ] 5.4 Extend `main_test.go`: exec smoke — list marks active, no profiles, unknown switch, model persists, usage exit 2

## Phase 6: Verification

- [ ] 6.1 Run `go test ./...`, `go vet`, `gofmt -l .`
- [ ] 6.2 Trace every REQ to a passing test; baseline 55 green; revert readiness
