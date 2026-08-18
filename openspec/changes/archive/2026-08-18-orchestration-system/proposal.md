# Proposal: Orchestration System

## Intent

kui users need to create and customize harness profiles as powerful as gentle-pi — with their own agents, tools, models, and orchestration rules. The current system has all the building blocks (profiles, tools, skills, agent definitions, sub-agents) but lacks the orchestration layer that ties them together into a cohesive multi-agent system.

**Goal**: Allow any user to create a custom orchestration profile in YAML, defining agents with restricted tools, model assignments, and delegation rules — and have kui execute it automatically.

## Current Gap

| Component | Status | Gap |
|-----------|--------|-----|
| Profile schema | ✅ Exists | No orchestration metadata |
| Agent definitions | ✅ Exists (Markdown) | No tool restrictions, no provider override |
| Sub-agent tool | ✅ Exists (process-based) | No in-process execution, no structured results |
| Skill system | ✅ Exists | No agent-skill binding |
| Tool registry | ✅ Exists | No per-agent filtering at spawn |
| Model routing | ✅ Exists (per-profile) | No per-agent override |
| Orchestration rules | ❌ Missing | No delegation, no gatekeeper, no dedup |
| Result structuring | ❌ Missing | Just `{output, exit_code, error}` |
| Error recovery | ❌ Missing | No retry, no fallback |

## Proposed Solution

### Phase 1: Enhanced Agent Definitions
Extend the existing Markdown agent definition format with orchestration metadata:

```yaml
---
name: worker
description: Implementation worker
tools:
  include: [read, write, edit, bash]
  exclude: []  # tool exclusion
model: balanced
thinking: medium
provider: openai  # optional provider override
max_iterations: 10
permissions:
  - pattern: "*.go"
    action: allow
  - pattern: ".env*"
    action: deny
---
```

### Phase 2: In-Process Agent Spawner
Replace process-based sub-agent execution with in-process spawning:
- Each sub-agent gets its own `core.Agent` instance
- Isolated conversation history
- Shared tool registry but filtered per agent definition
- Structured result type with conversation history and token accounting

### Phase 3: Orchestration Tool
Add an `orchestrate` tool that the parent agent can use:
- Spawn specialized agents (explore, worker, reviewer)
- Fan-out/fan-in for parallel work
- Chain/pipeline for sequential work
- Conditional routing based on content

### Phase 4: Delegation Rules
Configurable rules in the profile:
- File count threshold for exploration delegation
- Write count threshold for worker delegation
- Context rule for preparation delegation

### Phase 5: Gatekeeper & Dedup
- Auto mode: validate phase results before next phase
- Session-scoped deduplication of agent spawns

## Scope

### In Scope
1. Enhanced agent definition schema (tools include/exclude, provider, permissions, max_iterations)
2. In-process agent spawner (wraps core.Agent, isolated conversation)
3. Structured result type (output, messages, tokens, cost)
4. Orchestration tool (fan-out/fan-in, chain, conditional)
5. Delegation rules (configurable thresholds)
6. Gatekeeper validation (auto mode)
7. Launch deduplication (session-scoped)
8. CLI commands (profile create, profile validate, profile agents)

### Out of Scope
1. Cross-process agent communication (future)
2. Distributed orchestration (future)
3. Agent marketplace (future)
4. Visual orchestration editor (future)

## Success Criteria

1. A user can create a profile YAML with agent definitions and orchestration rules
2. The `orchestrate` tool can spawn agents with restricted tools and model overrides
3. Results are structured with conversation history and token accounting
4. Delegation rules automatically route work to appropriate agents
5. Gatekeeper validates results in auto mode
6. No duplicate agent spawns within a session

## Risks

1. **In-process isolation**: Sub-agents share the same process — memory leaks or panics affect parent
   - Mitigation: Use goroutine isolation, recover from panics, limit iterations
2. **Tool conflicts**: Multiple agents writing to same files
   - Mitigation: File-level locking, conflict detection
3. **Cost explosion**: Parallel agents can consume tokens rapidly
   - Mitigation: Per-agent token budgets, cost tracking
4. **Complexity**: Orchestration adds cognitive load
   - Mitigation: sensible defaults, simple profile format, good error messages

## Dependencies

1. Existing profile system (✅ complete)
2. Existing tool registry (✅ complete)
3. Existing skill system (✅ complete)
4. Existing agent definitions (✅ complete, needs enhancement)
5. Existing sub-agent tool (✅ complete, needs in-process alternative)
