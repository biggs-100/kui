# Proposal: Profile System — First-Class Profiles with Hot Switching

## Intent

kui is a fixed-session runtime: one prompt, one tool set, one provider, one system prompt. The vision: profiles as first-class units — per-profile model, tools, skills, prompt, permissions — hot-swappable within the SAME session and easy to customize via profile.yaml. This change delivers the runtime + CLI foundation; the TUI comes next.

## Scope

### In Scope
- Profile runtime: profile.yaml {name, model, system_prompt ref, tools, skills, permissions} + SYSTEM.md prompt body
- Manager: layered resolver (global → project → profile), active profile, hot switch queued between turns, same history
- Per-profile model persistence: map[profile]modelRef stored in .kui/
- Permissions: allow/ask/deny rulesets, wildcards, last-rule-wins; "*": "deny" hides tools from the model request
- Skills: index + trigger matching, available-skills list injected in system prompt, full SKILL.md loaded only on invocation; layered dirs
- Steering/follow-up dual queues (steer each turn, followUp on stop; QueueMode all|one-at-a-time)
- CLI: kui profile list|switch|model

### Out of Scope
- Bubble Tea TUI + TAB switching (next change)
- MCP servers, plugins, custom commands
- Interactive ask-permission prompts (deny/allow only)

## Capabilities

> Contract for sdd-spec: each new capability becomes `openspec/specs/<name>/spec.md`; each modified one gets a delta spec in this change.

### New Capabilities
- `profile-runtime`: profile model, loader, resolver, hot switch, model memory
- `profile-permissions`: rulesets, wildcards, last-rule-wins, tool hiding
- `profile-skills`: layered discovery, index + triggers, on-demand load
- `profile-cli`: profile subcommands on cmd/kui
- `steering-followup`: dual queues, two-level loop

### Modified Capabilities
- `agent-loop`: gains inner (tools + steering) and outer (follow-up) levels; switch applied between turns; profile-context marker message
- `agent-cli`: profile subcommands; per-profile model resolution at startup

## Approach

Core stays stdlib-only hexagonal: agent-loop restructures around a steering port (two pending-message queues); profile, permission, and skill ports added; I/O stays in adapters. Adapters: profile.yaml loader (pinned yaml lib), permission evaluator, indexed skill loader, model-memory store (.kui/), existing OpenAI-compatible provider via generic factory. Agent wrapper owns active profile + queues; a switch arrives via steering queue, applies between turns, and inserts a profile-context marker. CLI keeps one Agent, same history.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/core` (loop.go, provider.go) | Modified | Two-level loop, steering/follow-up, new ports |
| `internal/agent` | New | Agent wrapper, active profile, queues |
| `internal/adapters/profile` | New | Loader, resolver, model memory |
| `internal/adapters/permissions` | New | Ruleset evaluator, tool hiding |
| `internal/adapters/skills` | New | Layered index, on-demand loader |
| `cmd/kui` (main.go) | Modified | profile subcommands, model resolution |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Loop restructure regresses 55 green tests | Med | Additive queues; termination contract untouched; full suite gating |
| "*": "deny" hides tools → user confusion | Med | Documented; CLI lists hidden tools per profile |
| Mid-turn switch corrupts tool state | Low | Switch only between turns, via steering queue |
| yaml dep leaks into stdlib-only core | Med | Pinned lib, adapter layer only; guard test enforces |

## Rollback Plan

Single-commit revert restores prior behavior — additive change, no data migration. User data (.kui/, profile dirs) is deletable to reset; removed code paths are simply ignored.

## Dependencies

- Foundation on main: agent-loop, agent-tools, provider-openai-compatible, agent-cli; 55 green tests
- yaml.v3 (adapter-only); no DI framework, no dynamic installs

## Success Criteria

- [ ] go test ./... green — existing 55 plus new coverage (loop, permissions, skills, switching)
- [ ] CLI hot switch changes active profile mid-session; history preserved, marker present
- [ ] Test proves "*": "deny" removes the tool from the provider request payload
- [ ] Test proves skill content loads only on invocation (index-only system prompt)
- [ ] Per-profile model choice persists across sessions
