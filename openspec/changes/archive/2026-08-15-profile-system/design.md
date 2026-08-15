# Design: Profile System — First-Class Profiles with Hot Switching

## Technical Approach

Additive two-level loop over a port-based profile runtime. `internal/core` stays stdlib-only (guard test): it gains `PendingQueue`, `PermissionEvaluator`, `ProfileManager` ports, `SystemMessages` seeding, and the two-level `Run` restructure. `internal/agent` owns the concrete queues, the active profile, and the `ProfileManager` implementation; adapters do all yaml/filesystem work (`yaml.v3` pinned in `adapters/profile` only). `Chat()` keeps its signature — provider model mutates via a new `Client.SetModel`. Queues drain between turns; switches apply via `ProfileManager.ApplySwitch`, which returns the new system-prompt + marker messages appended to history (never mid-tool-call, never removing history). CLI resolves the active profile at startup and wires `profile list|switch|model`.

## Architecture Decisions (D14+, continuing the D1–D13 ADR table from agent-foundation)

| ADR | Decision | Options | Tradeoff | Choice |
|-----|----------|---------|----------|--------|
| D14 | Loop restructure | pi-style streamFn event loop vs additive queue fields | streamFn is the endgame but breaks provider/loop tests | Keep `Chat(ctx, msgs, tools)`; add nil-safe `Steering`/`FollowUp` fields + two-level `Run`; REQ-LOOP-1..4 untouched |
| D15 | Tool hiding | Provider decorator vs registry rebuild vs loop-level filter | decorator hides payload proof; rebuild loses defense-in-depth errors | Loop filters `Tools.List()` via `PermissionEvaluator.Filter` before Chat (REQ-PERM-3 payload) and guards dispatch with `Allow` → `PermissionError` (REQ-PERM-4) |
| D16 | Switch application | Wrapper pre/post hooks vs in-loop port call | hooks can't prove between-turn semantics | Loop calls `Profiles.ApplySwitch(ctx, name)` during steering drain; returns `[]Message` (new system prompt + marker) appended to history; also rebuilds registry subset from profile `tools` |
| D17 | Per-profile model | New client per switch vs `SetModel` vs Chat signature change | new client loses state; signature change breaks tests | `openai.Client.SetModel(model)` — additive; construction behavior unchanged |
| D18 | CLI switch semantics (resolves open question) | Session-start activation only vs steering-queue only vs **dual path** | standalone `switch` needs an observable outcome; queue-only is unusable without a prompt | `kui profile switch <name>` validates + persists `.kui/active` (session-start activation, visible in `profile list`); `kui profile switch <name> -- <prompt>` additionally enqueues a steering switch so the loop applies it mid-run with marker — proving the loop wiring end-to-end. `KUI_HOME` env overrides the global config dir for hermetic tests |
| D19 | Queue ownership | Queues in core vs in wrapper | core-owned leaks concurrency state into domain | Concrete `PendingMessageQueue` (mutex, `QueueMode` all|one-at-a-time) in `internal/agent`; core drains via `PendingQueue` port — TUI-ready |
| D20 | Persistence | yaml state vs JSON/text store | yaml in state dir drags the dependency everywhere | `.kui/models.json` (`map[profile]model`) + `.kui/active` (text) via `adapters/store` |
| D21 | Skills indexing | Parse frontmatter from SKILL.md vs separate metadata | index must build without reading bodies (REQ-SKILL-2) | Per-skill dir: `skill.yaml` (name/description/triggers) + `SKILL.md` body; `Load` reads body only at invocation (REQ-SKILL-3) |

## Data Flow

```
cmd/kui ── agent.Run(prompt)
   resolve active (.kui/active) → profile → SYSTEM.md + skills index → SystemMessages
   ▼
core.Run iteration:
  drain Steering ── user msgs appended; switch msgs → ApplySwitch → [system, marker] appended
  tools = Permissions.Filter(Registry)
  Chat(messages, tools) ── tool calls? → Allow? → Execute → append results → next iteration
                            └ no calls ── FollowUp.Drain() empty? → return answer
                                          └ non-empty → append user msgs, continue (budget still counts)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/core/queue.go` | Create | `PendingMessage`, `PendingQueue` port, `QueueMode` |
| `internal/core/profile.go` | Create | `ProfileManager` port, `ModelMemory` port |
| `internal/core/permissions.go` | Create | `PermissionEvaluator` port |
| `internal/core/loop.go` | Modify | New fields; two-level `Run`; steering drain + switch apply + filter + dispatch guard |
| `internal/core/provider.go` | Modify | `RoleSystem` constant (marker/system messages) |
| `internal/core/errors.go` | Modify | `UnknownProfileError`, `PermissionError`, `ProfileActivationError` (names file), `SkillLoadError`, `StoreError` |
| `internal/agent/agent.go` | Create | Wrapper: active profile, queue refs, `Run` wiring, `LoadSkill` |
| `internal/agent/queues.go` | Create | Mutex `PendingMessageQueue`, `Enqueue`/`Drain` per mode |
| `internal/agent/profile_manager.go` | Create | `ApplySwitch`: resolve+load profile, SYSTEM.md, rebuild evaluator/registry, `SetModel`, return messages |
| `internal/adapters/profile/loader.go` | Create | yaml.v3 loader, discovery, pure layered resolver (global→project→profile) |
| `internal/adapters/permissions/ruleset.go` | Create | `Flatten`, `Evaluate` (findLast, `path.Match` wildcards), Ask→Deny, `Filter` |
| `internal/adapters/skills/index.go` | Create | Layered discovery, `skill.yaml` index, trigger `Match`, body `Load` |
| `internal/adapters/store/store.go` | Create | `models.json` + `active` under `.kui/`, `KUI_HOME` |
| `internal/adapters/providers/openai/client.go` | Modify | `SetModel` |
| `cmd/kui/main.go` | Modify | Subcommands, startup resolution, `--` prompt form, usage text |

## Interfaces / Contracts

```go
// core — ports (stdlib-only)
type PendingQueue interface { Drain() []PendingMessage }
type PendingMessage struct { Content string; SwitchProfile string }
type PermissionEvaluator interface { Allow(name string) bool; Filter(tools []Tool) []Tool }
type ProfileManager interface { ApplySwitch(ctx context.Context, name string) ([]Message, error) }
type ModelMemory interface { Get(profile string) (string, bool); Set(profile, model string) error }
// openai adapter — additive
func (c *Client) SetModel(model string)
// permissions adapter
func Flatten(layers ...[]Rule) *Ruleset // defaults → config → profile; last matching rule wins
// Resolution chain (REQ-CLI-4): ModelMemory → profile.yaml → project config → global config → OPENAI_MODEL → gpt-4o-mini
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit (core) | Drain all/one-per-turn; follow-up at stop; budget + queued msgs → IterationLimitError; tool failure skips injection; switch between turns + marker; multi-switch last-wins; filter applied to Chat's tools; denied dispatch → PermissionError | Extend `loop_test.go` with fake queue/manager/provider recording the tools slice |
| Unit (adapters) | Ruleset last-wins/wildcards/empty/ask→deny/unknown tool; skills collision, index-without-body, load-on-demand, missing body file; store persist/restore; resolver precedence | Package tests per adapter |
| Payload | REQ-PERM-3: JSON `tools` array excludes denied tool | httptest capturing body in `client_test.go` + permissions integration |
| CLI smoke | `list` marks active, exit 0; `switch nope` → stderr names it, non-zero; `model` persists; missing args → usage, exit 2 | Exec-binary pattern from `main_test.go`, temp cwd + `KUI_HOME` |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary is introduced or modified. `bash.go` and its D12 subprocess hardening are untouched; permissions gate dispatch only, proven side-effect-free with fake tools at loop level (REQ-PERM-4).

## Migration / Rollout

No migration. All new fields nil-safe (55 existing tests untouched); `.kui/` files created on first use; `yaml.v3` added to `go.mod`, adapter-only, guard test gates the core boundary. Rollback: single-commit revert.

## Open Questions

None — the CLI switch semantics open question is resolved by D18 (dual path).
