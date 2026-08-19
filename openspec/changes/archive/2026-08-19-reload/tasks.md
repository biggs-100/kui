# Tasks: Hot Reload (reload command)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~650 (A: 150, B: 300, C: 200) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Slice A) → PR 2 (Slice B) → PR 3 (Slice C) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| A | Manager lock + Agent setters | PR 1 | go test -race ./internal/agent/... | Real -race: concurrent ApplySwitch/Reload | Revert agent.go, profile_manager.go |
| B | internal/runtime lifecycle + extAPI | PR 2 | go test ./internal/runtime/... && go build ./... | Temp-dir integration: new skill+profile, MCP reconnect, failed-build keeps old | Delete internal/runtime/, restore entrypoints |
| C | Controller tracking + /reload TUI | PR 3 | go test -race ./internal/tui/... | teatest TUI: /reload slash, status, cancel-and-wait | Revert controller/app/chat, drop /reload |

## Phase 1: Slice A — Manager race fix + Agent setters

- [x] 1.1 RED `internal/agent/profile_manager_test.go`: -race concurrent ApplySwitch/Reload (REQ-RELOAD-9); re-apply active profile; failed re-apply keeps old registry (REQ-RELOAD-10)
- [x] 1.2 RED `internal/agent/agent_test.go`: setters swap skills/provider; nil SetHooks parity (REQ-RELOAD-19/20)
- [x] 1.3 GREEN `internal/agent/profile_manager.go`: sync.Mutex over registry/ruleset/active/model; `Reload(full)` swap + re-apply; UnknownProfileError clears active, other errors keep old registry
- [x] 1.4 GREEN `internal/agent/agent.go`: hooks field + SetSkills/SetProvider/SetHooks; `Run` wires `loop.Hooks = a.hooks` (nil safe)
- [x] 1.5 REFACTOR: gofmt, go vet, `go test -race ./internal/agent/...` green

## Phase 2: Slice B — internal/runtime package

- [x] 2.1 RED `internal/runtime/runtime_test.go`: build snapshot + error propagation (REQ-RELOAD-1); reload picks up new skill+profile, MCP reconnect (REQ-RELOAD-3); failed build keeps old state (REQ-RELOAD-4); Close idempotent, ShutdownAll once (REQ-RELOAD-5/18); extAPI tool+hook registration, duplicate error (REQ-RELOAD-16/17)
- [x] 2.2 GREEN `internal/runtime/runtime.go`: Runtime{Store,Loader,Agent,Provider,Manager,Skills,Full,MCP,Hooks,Profiles}; `Build(ctx,cfg)`: provider → loader.Discover → skills index → MCP ConnectAll → registry → extAPI LoadAll → hooks → steering
- [x] 2.3 GREEN `internal/runtime/runtime.go`: `Reload(ctx)` pi-order 10 steps: ShutdownAll → MCP.Shutdown → provider → Discover → skills → registry rebuild → Manager.Reload(full) → SetModel → SetHooks → steering re-seed; returns ReloadResult
- [x] 2.4 GREEN `internal/runtime/extapi.go`: concrete ExtensionAPI (registry, hook registry, command slice)
- [x] 2.5 GREEN `cmd/kui/main.go` + `internal/tui/run.go`: replace inline composition with `runtime.Build` (REQ-RELOAD-2); runPrompt/runTUI delegate
- [x] 2.6 REFACTOR: `go test ./...`, `go build ./...`, gofmt, go vet

## Phase 3: Slice C — TUI integration

- [x] 3.1 RED `internal/tui/controller_test.go`: run lifecycle tracked; cancel-and-wait with blocking fake runner; cancel suppresses error, genuine errors show (REQ-RELOAD-6/7/8/13); no-reloader path (REQ-RELOAD-14); profile refresh, active preserved/fallback (REQ-RELOAD-15)
- [x] 3.2 RED `internal/tui/app_test.go` + `internal/tui/views/chat_test.go`: `/reload` reloads, not prompts (REQ-RELOAD-11); status "reloading…" / "reload complete: N skills" / error (REQ-RELOAD-12)
- [x] 3.3 GREEN `internal/tui/controller.go`: running/cancel/runDone under mu; SubmitPrompt cancellable ctx; `Reload()` cancel-and-wait then invoke reloader; SetReloader port; Reloader/ReloadResult types; reloadStartMsg/reloadDoneMsg; re-detect SetModeler from runner.Provider()
- [x] 3.4 GREEN `internal/tui/app.go`: slash parse in handleKey Enter branch; `/reload` → ctrl.Reload(), never SubmitPrompt; reload msgs → ChatModel.SetStatus; profiles refresh preserving active by name, first-profile fallback
- [x] 3.5 GREEN `internal/tui/views/chat.go`: SetStatus + status line render
- [x] 3.6 GREEN `internal/tui/run.go`: `SetReloader(runtime.Reload)`
- [x] 3.7 REFACTOR: `go test -race ./...`, gofmt, go vet; manual TUI `/reload` smoke test
