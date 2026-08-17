# Design: Hot Reload (reload command)

## Technical Approach

New `internal/runtime` package owns lifecycle composition. `Build(ctx, cfg)` assembles a complete runtime snapshot; `Reload(ctx)` cancels active runs (via the controller), re-reads disk state, and swaps only on a clean build; `Close()` tears down. Both `runPrompt` and `tui.Run` delegate to `runtime.Build`, killing the duplicated composition. The controller gains run tracking (cancel-and-wait), a `/reload` slash command, and a status line; `agent.Manager` becomes lock-guarded with `Reload(full)`.

## Architecture Decisions

| # | Option | Tradeoff | Decision |
|---|--------|----------|----------|
| D1 | Extract composition into `internal/runtime` vs. keep duplicated build | One path fixes drift risk (REQ-RELOAD-2) | Extract. `Runtime{Store, Loader, Agent, Provider, Manager, Skills, Full, MCP, Hooks, Profiles}`. Loader is stateless (reads disk per Resolve/Discover), so one instance survives reloads. |
| D2 | Mutate in place vs. build-fresh-then-commit | In-place leaks partial state on error | Build fresh locals; only `Manager.Reload(full)` mutates (atomic under lock); remaining steps (SetModel/hooks/steering) are infallible. Failed build returns error, old runtime stays active (REQ-RELOAD-4). Teardown-first ordering (REQ-RELOAD-18, pi parity) means a failed reload leaves old MCP/extensions shut down — accepted degradation; user retries. |
| D3 | Reload ordering | Order fixes stale-state windows | 1) controller cancels run and waits → 2) `extensions.ShutdownAll()` → 3) old `MCP.Shutdown()` → 4) `cfg.Client()` recreates provider (re-reads env) → 5) `loader.Discover()` → 6) rebuild skills index; `Agent.SetSkills` → 7) rebuild registry (builtin tools → new MCP ConnectAll → tools → extAPI `LoadAll`); `Manager.Reload(full)` → 8) SetModel via resolver chain → 9) `Agent.SetHooks(newHooks)` → 10) steering re-seed (`SwitchProfile` + `SystemMessages`). Steps 2–3 are unrecoverable teardown; 4–7 failures return error with old runtime active. |
| D4 | `Manager` concurrency | Unsynchronized `ApplySwitch` races reload (REQ-RELOAD-9) | `sync.Mutex` guards `registry/ruleset/active/model/full`. `ApplySwitch`/`Reload` hold lock across full mutation; `Registry/Ruleset/Active/Model` read-lock. `Reload(full)` swaps `full` and re-applies active profile; `UnknownProfileError` (profile deleted) clears active and succeeds (REQ-RELOAD-15 fallback); other resolve errors keep old registry and return error (REQ-RELOAD-10). |
| D5 | Controller run tracking | Reload during a run loses/corrupts state | Add `running/cancel/runDone` (under existing `mu`). `SubmitPrompt` creates a cancellable context, stores cancel + done channel. `Reload()`: if running, capture cancel+done under lock, release, cancel, wait on done; then invoke reloader. Canceled runs suppress error emit (REQ-RELOAD-8); genuine errors still display. After reload, re-detect `SetModeler` from `runner.Provider()` (new provider, REQ-RELOAD-19). |
| D6 | TUI `/reload` command | Slash text would hit the agent | `App.handleKey` Enter branch: input starting with `/` is parsed; `/reload` → `ctrl.Reload()`, never `SubmitPrompt`. `reloadStartMsg` → `chat.SetStatus("reloading…")`; `reloadDoneMsg` → "reload complete: N skills" or the error (REQ-RELOAD-12). Controller refreshes `profiles` from `ReloadResult.Profiles`, preserving active by name, first-profile fallback (REQ-RELOAD-15). |
| D7 | Extension wiring | Extensions never initialized in production | Concrete `extAPI` (registry + hook registry + command slice) in `internal/runtime/extapi.go`; `LoadAll` in `Build` after MCP tools; `ShutdownAll` before rebuild and on `Close`. ShutdownAll-first invariant prevents double-Init (LoadAll reassigns `loaded` on success); Build failure keeps prior runtime active (REQ-RELOAD-17). |
| D8 | Agent setters | Hooks never fire; Agent not swappable | Add `hooks *core.HookRegistry` field; `SetSkills/SetProvider/SetHooks`; `Run` wires `loop.Hooks = a.hooks` (nil keeps behavior identical, REQ-RELOAD-20). |

## Data Flow

```
TUI /reload ──> ctrl.Reload()
                 ├─ running? ── cancel ctx ──> wait runDone (suppress err)
                 └─> reloader(ctx) = runtime.Reload
                       ShutdownAll → MCP.Shutdown → provider/skills/profiles rebuild
                       └─> newFull registry → Manager.Reload(full) → SetModel → SetHooks → steering re-seed
                       └─> ReloadResult{Profiles, Skills, Err}
                 └─> refresh profiles → emit reloadDoneMsg → App → ChatModel.SetStatus
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/runtime/runtime.go` | Create | `Runtime`, `Config`, `Build`, `Reload`, `Close` |
| `internal/runtime/extapi.go` | Create | Concrete `core.ExtensionAPI` |
| `internal/tui/run.go` | Modify | Replace composition with `runtime.Build`; `Wiring` → `runtime.Config`; `SetReloader` |
| `cmd/kui/main.go` | Modify | `runPrompt`/`runTUI` use `runtime.Build` |
| `internal/tui/controller.go` | Modify | Run tracking, `Reload()`, `SetReloader`, `Reloader`/`ReloadResult`, reload msgs |
| `internal/tui/app.go` | Modify | Slash parse in `handleKey`, reload msg handling |
| `internal/tui/views/chat.go` | Modify | `SetStatus` + status render |
| `internal/agent/agent.go` | Modify | `hooks` field, setters, `loop.Hooks` wiring |
| `internal/agent/profile_manager.go` | Modify | Mutex, `Reload(full)` |
| `internal/adapters/extensions/registry.go` | Modify | None required (ShutdownAll-first invariant); defensive `loaded=nil` at `LoadAll` entry optional |

## Interfaces / Contracts

```go
type Config struct {
    ProfileRoot, ProjectDir, ConfigRoot string
    Client   func() (core.Provider, error) // re-reads env on reload
    MaxIter  int
}
type ReloadResult struct { Profiles []string; Skills int; Err error }
type Reloader  func(ctx context.Context) ReloadResult
func Build(ctx context.Context, cfg Config) (*Runtime, error)
func (r *Runtime) Reload(ctx context.Context) ReloadResult
func (r *Runtime) Close() error
func (m *Manager) Reload(full *core.Registry) error
func (c *Controller) SetReloader(r Reloader)   // port; controller never imports runtime
func (c *Controller) Reload()
func (a *Agent) SetSkills(*skills.Index); SetProvider(core.Provider); SetHooks(*core.HookRegistry)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Manager mutex + `Reload(full)` (swap, re-apply, UnknownProfile fallback, failed re-apply) | `go test -race`; concurrent ApplySwitch/Reload |
| Unit | Agent setters; `loop.Hooks` wiring (nil + non-nil) | Direct calls; nil-hooks parity test |
| Unit | Controller run tracking, cancel-and-wait, cancel suppresses error, no-reloader path | Fake runner with blocking Run + cancelable ctx |
| Unit | Slash parse; ChatModel status render | `app_test.go`, `views/chat_test.go` |
| Integration | `runtime.Build`/`Reload` over temp dirs: new skill+profile picked up, MCP reconnects, failed build keeps old state | Real loader/skills/store; fake extension registering tool+hook; fake MCP factory |
| E2E | `go test -race ./...` clean; manual `/reload` in TUI | Success criteria |

## Threat Matrix

| Boundary | Applicability | Reason / Design response |
|---|---|---|
| Documentation-like paths | N/A | No doc-path classification introduced |
| Git repository selection | N/A | No VCS automation |
| Commit state | N/A | No commit logic |
| Push state | N/A | No push logic |
| PR commands | N/A | No PR automation |

No routing, shell-command, or executable-classification boundary is added. MCP subprocess spawning is pre-existing and unchanged (commands/cwd/env come from `mcp.yaml` verbatim); reload only calls existing `ConnectAll`/`Shutdown`. Process-integration edge (reload mid-run) is covered by context-cancellation cancel-and-wait (REQ-RELOAD-6/7) — RED tests in controller unit layer.

## Migration / Rollout

No migration. Rollback per proposal: remove `internal/runtime/`, restore entrypoint composition, drop `/reload`; keep the `Manager` lock.

## Open Questions

- None blocking. (SetModeler refresh via `runner.Provider()` re-detection rather than an explicit field swap — verify in tests.)