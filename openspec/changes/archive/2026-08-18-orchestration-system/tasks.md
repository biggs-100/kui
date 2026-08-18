# Tasks: Orchestration System

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1200–1500 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 → PR 4 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

---

## Phase 1: Agent Definition Schema (PR 1) ✅

- [x] 1.1 RED — Test agent definition parsing
- [x] 1.2 GREEN — Implement AgentDef struct + parser
- [x] 1.3 RED — Test tool include/exclude filtering
- [x] 1.4 GREEN — Implement `FilterTools`
- [x] 1.5 RED — Test agent definition resolution chain
- [x] 1.6 GREEN — Implement `AgentRegistry` with resolution
- [x] 1.7 RED — Test built-in agents
- [x] 1.8 GREEN — Define built-in agents

**PR 1 Total**: 242 code + 468 tests = 710 lines

---

## Phase 2: In-Process Spawner (PR 2) ✅

### Task 2.1: RED — Test spawner creates isolated agent
- [x] **Test**: `TestSpawnerCreatesAgent` — spawn returns valid result
- **Files**: `internal/orchestration/spawner_test.go`
- **Estimated**: 40 lines

### Task 2.2: GREEN — Implement `Spawner` struct
- [x] **Code**: `Spawner` struct, `NewSpawner(provider, registry, profiles)`
- **Files**: `internal/orchestration/spawner.go`
- **Estimated**: 50 lines

### Task 2.3: RED — Test tool filtering at spawn
- [x] **Test**: `TestSpawnerFiltersTools` — spawned agent has correct tools
- **Files**: `internal/orchestration/spawner_test.go`
- **Estimated**: 30 lines

### Task 2.4: GREEN — Implement `Spawn(ctx, req)`
- [x] **Code**: `Spawn()` with tool filtering, agent creation, execution
- **Files**: `internal/orchestration/spawner.go`
- **Estimated**: 100 lines

### Task 2.5: RED — Test structured result
- [x] **Test**: `TestSpawnResult` — output, messages, tokens, duration
- **Files**: `internal/orchestration/spawner_test.go`
- **Estimated**: 30 lines

### Task 2.6: GREEN — Implement `SpawnResult` struct
- [x] **Code**: `SpawnResult` struct with fields
- **Files**: `internal/orchestration/spawner.go`
- **Estimated**: 30 lines

### Task 2.7: RED — Test panic recovery
- [x] **Test**: `TestSpawnerPanicRecovery` — agent panic returns error
- **Files**: `internal/orchestration/spawner_test.go`
- **Estimated**: 25 lines

### Task 2.8: GREEN — Add panic recovery wrapper
- [x] **Code**: `recover()` wrapper around agent execution
- **Files**: `internal/orchestration/spawner.go`
- **Estimated**: 20 lines

### Task 2.9: RED — Test model/thinking override
- [x] **Test**: `TestSpawnerModelOverride` — override from request
- **Files**: `internal/orchestration/spawner_test.go`
- **Estimated**: 25 lines

### Task 2.10: GREEN — Apply model/thinking overrides
- [x] **Code**: Override logic in `Spawn()`
- **Files**: `internal/orchestration/spawner.go`
- **Estimated**: 20 lines

**PR 2 Total**: 257 lines (99 code + 158 tests)

---

## Phase 3: Orchestrator Tool (PR 3) ✅

### Task 3.1: RED — Test tool schema
- [x] **Test**: `TestToolSchema` — valid JSON schema, required fields
- **Files**: `internal/orchestration/tool_test.go`

### Task 3.2: GREEN — Implement `Tool` struct + `Schema()`
- [x] **Code**: `Tool` struct, `Schema()` method, `NewTool()`
- **Files**: `internal/orchestration/tool.go`

### Task 3.3: RED — Test spawn operation
- [x] **Test**: `TestToolSpawn` — single agent spawn
- **Files**: `internal/orchestration/tool_test.go`

### Task 3.4: GREEN — Implement `Execute()` for spawn
- [x] **Code**: `Execute()` with spawn operation, `executeSpawn()`
- **Files**: `internal/orchestration/tool.go`

### Task 3.5: RED — Test fan-out operation
- [x] **Test**: `TestToolFanOut`, `TestToolFanOutParallel` — parallel spawns
- **Files**: `internal/orchestration/tool_test.go`

### Task 3.6: GREEN — Implement fan-out with goroutines
- [x] **Code**: Fan-out with WaitGroup, `executeFanOut()`
- **Files**: `internal/orchestration/tool.go`

### Task 3.7: RED — Test chain operation
- [x] **Test**: `TestToolChain`, `TestToolChainFeedsOutput` — sequential with output feeding
- **Files**: `internal/orchestration/tool_test.go`

### Task 3.8: GREEN — Implement chain operation
- [x] **Code**: Chain with sequential execution, `executeChain()`
- **Files**: `internal/orchestration/tool.go`

### Task 3.9: RED — Test result aggregation
- [x] **Test**: `TestAggregateMerge/Summary/Select/Default` + edge cases
- **Files**: `internal/orchestration/aggregator_test.go`

### Task 3.10: GREEN — Implement `ResultAggregator`
- [x] **Code**: `ResultAggregator` with merge/summary/select modes
- **Files**: `internal/orchestration/aggregator.go`

### Task 3.11: RED — Test dedup integration
- [x] **Test**: `TestToolDedup` + `TestDedupFirstCall/SecondCall/Reset/Concurrent` etc.
- **Files**: `internal/orchestration/dedup_test.go`, `tool_test.go`

### Task 3.12: GREEN — Add dedup to tool
- [x] **Code**: `LaunchDedup` with result caching, integrated in `Execute()`
- **Files**: `internal/orchestration/dedup.go`, `tool.go`

**PR 3 Total**: 1029 lines (392 code + 637 tests)

---

## Phase 4: Delegation Rules + Gatekeeper (PR 4) ✅

### Task 4.1: RED — Test delegation rules
- [x] **Test**: `TestDelegationRules` — explore/write/context thresholds
- **Files**: `internal/orchestration/rules_test.go`

### Task 4.2: GREEN — Implement `Rules` struct
- [x] **Code**: `Rules` struct, `ShouldDelegate(action, count)`
- **Files**: `internal/orchestration/rules.go`

### Task 4.3: RED — Test gatekeeper validation
- [x] **Test**: `TestGatekeeperValidate` — empty output, error, token budget
- **Files**: `internal/orchestration/gatekeeper_test.go`

### Task 4.4: GREEN — Implement `Gatekeeper` struct
- [x] **Code**: `Gatekeeper` struct, `Validate(result)`, `ShouldRetry(result, attempt)`
- **Files**: `internal/orchestration/gatekeeper.go`

### Task 4.5: RED — Test engine wiring
- [x] **Test**: `TestEngineWireup` — all components connected
- **Files**: `internal/orchestration/engine_test.go`

### Task 4.6: GREEN — Implement `Engine` struct
- [x] **Code**: `Engine` struct, `NewEngine(profile)`, wires all components
- **Files**: `internal/orchestration/engine.go`

### Task 4.7: RED — Test profile YAML integration
- [x] **Test**: `TestProfileOrchestration` — load orchestration config from profile
- **Files**: `internal/orchestration/engine_test.go`

### Task 4.8: GREEN — Add orchestration to profile schema
- [x] **Code**: `Orchestration` field in profile YAML
- **Files**: `internal/adapters/profile/loader.go`

### Task 4.9: RED — Test CLI commands
- [x] **Test**: `TestCLIProfileAgents` — list agents command
- **Files**: `cmd/kui/main_test.go`

### Task 4.10: GREEN — Add CLI subcommands
- [x] **Code**: `kui profile agents`, `kui profile validate`
- **Files**: `cmd/kui/main.go`

**PR 4 Total**: ~350 lines (175 code + 175 tests)

---

## Summary

| PR | Phase | Components | Lines |
|----|-------|------------|-------|
| 1 | Agent Definition | agentdef.go, agentdef_test.go | ~350 |
| 2 | In-Process Spawner | spawner.go, spawner_test.go | ~370 |
| 3 | Orchestrator Tool | tool.go, aggregator.go, dedup.go + tests | ~455 |
| 4 | Rules + Engine | rules.go, gatekeeper.go, engine.go + tests | ~350 |
| **Total** | | | **~1525** |

### Dependencies

```
PR 1 (agentdef) → PR 2 (spawner) → PR 3 (tool) → PR 4 (engine)
```

### Implementation Order

1. **PR 1**: Agent definition schema — foundation for everything
2. **PR 2**: In-process spawner — uses agent definitions
3. **PR 3**: Orchestrator tool — uses spawner + aggregation
4. **PR 4**: Rules + engine — ties everything together
