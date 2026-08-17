# Delta for agent-setters

Targets main spec: `openspec/specs/agent-loop/spec.md`. Existing loop requirements REQ-LOOP-1..15 remain valid unchanged; this delta adds Agent-wrapper setters and hook wiring that production never used.

## ADDED Requirements

### Requirement: REQ-RELOAD-19 — Agent Setters

`agent.Agent` MUST expose `SetSkills(*skills.Index)`, `SetProvider(core.Provider)`, and `SetHooks(*core.HookRegistry)` so the runtime can swap state on reload without recreating the Agent.

#### Scenario: Skills swapped after reload

- GIVEN an Agent with an old skills index
- WHEN SetSkills(newIndex) is called
- THEN SystemMessages reflects the new index

#### Scenario: Provider swapped after reload

- GIVEN an Agent with a provider
- WHEN SetProvider(newProvider) is called
- THEN subsequent runs use the new provider
- AND the controller's StreamingProvider detection sees the new provider

#### Scenario: Nil hooks setter is safe

- GIVEN SetHooks(nil)
- WHEN the agent runs
- THEN the loop behaves as before
- AND no hooks fire

### Requirement: REQ-RELOAD-20 — Agent.Run Wires Hooks

`Agent.Run` MUST wire `loop.Hooks` from the agent's hook registry. Today the `HookRegistry` field is never wired, so hooks never fire in production. A nil registry MUST keep the loop's behavior identical (REQ-LOOP-12).

#### Scenario: Hooks fire in production runs

- GIVEN an Agent with a configured hook registry
- WHEN Run executes
- THEN loop.Hooks is set to the registry
- AND registered hooks fire at their lifecycle points

#### Scenario: No hooks — unchanged behavior

- GIVEN an Agent without a hook registry
- WHEN Run executes
- THEN loop.Hooks is nil
- AND all existing loop tests pass unmodified