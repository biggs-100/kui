# Proposal: Hot Reload (reload command)

## Intent

kui needs a full restart to pick up config, profile, skill, or extension changes, dropping the TUI session, active context, and MCP connections. A `/reload` command refreshes runtime state without exiting (parity: pi's `reload`).

## Scope

### In Scope
- New `internal/runtime` package: `Build` / `Reload` / `Close`
- Build-new-then-swap — failed reload keeps old state
- TUI `/reload` command, status, cancel-and-wait
- Fix `agent.Manager` race (unsynchronized `ApplySwitch`)
- Consolidate build composition in `runPrompt` and `tui.Run`
- Wire `extensions.ShutdownAll` + MCP `Shutdown` into production reload

### Out of Scope
- Filesystem watching / auto-reload; process restart
- Implementing the concrete `ExtensionAPI` (dead code stays dead)

## Capabilities

### New Capabilities
- `runtime-lifecycle`: Build/Reload/Close orchestration, cancel-and-wait, build-new-then-swap

### Modified Capabilities
- `agent-loop`: `Manager.Reload` + synchronized `ApplySwitch`
- `extension-system`: `ShutdownAll` in prod path
- `mcp-manager`: `Shutdown` for teardown/recreate
- `tui-app`: `/reload` command, status, cancel-and-wait
- `steering-followup`: re-seed after reload
- `profile-runtime`: re-discover profiles

## Approach

`internal/runtime` owns the lifecycle: `Build` assembles a full runtime snapshot (providers, profiles, skills, MCP, hooks, steering); `Reload` cancels-and-waits active runs, tears down MCP/extensions, re-runs discovery and build, swapping only on a clean build.

Sequence: cancel-and-wait → teardown (`ShutdownAll`, MCP `Shutdown`) → rebuild (provider, profiles, skills, registry) → `Manager.Reload` + `SetModel` → hooks re-wire → steering re-seed.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/runtime/` | New | Lifecycle orchestration |
| `internal/agent/agent.go` | Modified | `Manager.Reload` |
| `internal/agent/profile_manager.go` | Modified | Mutex around `ApplySwitch` |
| `internal/adapters/extensions/registry.go` | Modified | `ShutdownAll` in prod path |
| `internal/mcp/manager.go` | Modified | `Shutdown` |
| `internal/tui/controller.go` | Modified | `/reload` command, status |
| `internal/tui/run.go` | Modified | `runtime.Build` |
| `cmd/kui/main.go` | Modified | `runPrompt` uses `runtime.Build` |
| `internal/adapters/skills/index.go` | Modified | Re-scan |
| `internal/adapters/profile/loader.go` | Modified | Re-discover |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Reload during active run loses context | Med | cancel-and-wait; user-initiated |
| Partial build leaves inconsistent state | Med | build-new-then-swap |
| `Manager` race on reload | High | mutex over `ApplySwitch`/`Reload` |
| TUI stream bypasses `agent.Run` | Med | shared `runtime.Build` |

## Rollback Plan

Remove `internal/runtime/`; restore original build composition in `runPrompt` and `tui.Run`; drop `/reload`; keep the `Manager` lock. No migration — reload never persists state.

## Dependencies

Internal: `extensions.ShutdownAll`, MCP `Shutdown`, `Manager.Reload`.

## Success Criteria

- [ ] `/reload` refreshes profiles/skills/provider without restart
- [ ] Failed reload keeps prior state and surfaces the error
- [ ] Active runs cancelled cleanly; TUI shows reload status
- [ ] `go test -race ./...` clean
- [ ] `runPrompt` and `tui.Run` share `runtime.Build`