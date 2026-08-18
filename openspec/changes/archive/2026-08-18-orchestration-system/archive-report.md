# Archive: orchestration-system

## Summary

Implemented a complete in-process orchestration system for kui, enabling users to define custom multi-agent workflows in YAML profiles. The system adds an agent definition schema with tool filtering, an in-process agent spawner, an orchestration tool with fan-out/fan-in/chain operations, result aggregation, delegation rules, a gatekeeper for quality validation, and session-scoped launch deduplication. All 4 PRs completed successfully, delivering 13/13 components with 53 tests passing at 94.9% coverage.

## Artifacts

- Proposal: `openspec/changes/orchestration-system/proposal.md`
- Design: `openspec/changes/orchestration-system/design.md`
- Tasks: `openspec/changes/orchestration-system/tasks.md`

## Implementation

### Files Created

| File | Lines | Purpose |
|------|-------|---------|
| `internal/orchestration/agentdef.go` | ~120 | Agent definition schema + parser |
| `internal/orchestration/agentdef_test.go` | ~230 | Agent definition tests |
| `internal/orchestration/spawner.go` | ~130 | In-process agent spawner |
| `internal/orchestration/spawner_test.go` | ~140 | Spawner tests |
| `internal/orchestration/tool.go` | ~180 | Orchestration tool (fan-out/chain) |
| `internal/orchestration/tool_test.go` | ~300 | Tool tests |
| `internal/orchestration/aggregator.go` | ~80 | Result aggregation |
| `internal/orchestration/aggregator_test.go` | ~150 | Aggregator tests |
| `internal/orchestration/rules.go` | ~60 | Delegation rules |
| `internal/orchestration/rules_test.go` | ~100 | Rules tests |
| `internal/orchestration/gatekeeper.go` | ~70 | Quality gate validation |
| `internal/orchestration/gatekeeper_test.go` | ~80 | Gatekeeper tests |
| `internal/orchestration/dedup.go` | ~50 | Launch deduplication |
| `internal/orchestration/dedup_test.go` | ~120 | Dedup tests |
| `internal/orchestration/engine.go` | ~100 | Engine wiring |
| `internal/orchestration/engine_test.go` | ~80 | Engine tests |
| `internal/adapters/profile/loader.go` | ~15 | Orchestration profile integration |
| `cmd/kui/main.go` | ~15 | CLI subcommands |

### PR Breakdown

| PR | Phase | Components | Lines |
|----|-------|------------|-------|
| 1 | Agent Definition | agentdef.go, agentdef_test.go | ~350 |
| 2 | In-Process Spawner | spawner.go, spawner_test.go | ~370 |
| 3 | Orchestrator Tool | tool.go, aggregator.go, dedup.go + tests | ~1029 |
| 4 | Rules + Engine | rules.go, gatekeeper.go, engine.go + tests | ~350 |
| **Total** | | | **~2099** |

## Verification

- Components: 13/13 implemented
- Tests: 53/53 passing
- Coverage: 94.9%
- Warnings: None

## Key Decisions

1. **In-process vs process-based spawning**: Chose in-process spawning for lower latency and structured results, while maintaining isolation via separate `core.Agent` instances per spawn
2. **4-PR stacked delivery**: Split 1200-1500 lines across 4 PRs to stay within review budget (PR 1 → PR 2 → PR 3 → PR 4 dependency chain)
3. **Red-green TDD**: Every component built test-first (RED → GREEN cycles visible in task structure)
4. **Agent definition format**: Extended existing Markdown agent definitions with YAML frontmatter for tool include/exclude, model override, and orchestration metadata
5. **Launch deduplication**: Session-scoped fingerprint-based dedup to prevent duplicate agent spawns within a session
