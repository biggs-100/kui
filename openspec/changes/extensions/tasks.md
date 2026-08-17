# Tasks: Extension System with Lifecycle Hooks

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 450–550 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Slice A → Slice B → Slice C (stacked-to-main) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| A | Core ports: Extension, HookRegistry, HookContext interfaces + impl | PR 1 | `go test ./internal/core/ -run TestHook` | `N/A — pure interface tests` | `internal/core/extension.go`, `hook_registry.go`, `hook_context.go` — remove files, no loop changes |
| B | Loop integration: emit hooks at 3 lifecycle points | PR 2 | `go test ./internal/core/ -run TestLoopHook` | `go run ./cmd/kui -- "hello"` (no hooks wired = same behavior) | `internal/core/loop.go` hook calls — revert lines, no new files |
| C | Discovery + example extension | PR 3 | `go test ./internal/adapters/extensions/...` | `go run ./cmd/kui -- "hello"` (blank import + example ext) | `internal/adapters/extensions/` dir + `cmd/kui/main.go` import — remove dir + import |

---

## Phase 1: Core Ports (internal/core)

- [x] 1.1 RED: `internal/core/extension_test.go` — test Extension interface compiles, Name/Init/Shutdown contract
- [x] 1.2 GREEN: `internal/core/extension.go` — create Extension interface (Name, Init, Shutdown) and ExtensionAPI interface (RegisterTool, RegisterHook, RegisterCommand)
- [x] 1.3 RED: `internal/core/hook_registry_test.go` — test Register order, Emit order, nil handler rejection, error short-circuit, HasHooks fast-path, nil-receiver safety
- [x] 1.4 GREEN: `internal/core/hook_registry.go` — HookRegistry struct with Register, Emit, HasHooks; nil-safe pointer receivers (D3, D7)
- [x] 1.5 RED: `internal/core/hook_context_test.go` — test message mutation, block/unblock, nil-safe Messages()
- [x] 1.6 GREEN: `internal/core/hook_context.go` — concrete hookContext implementing HookContext interface (D4)
- [x] 1.7 RED: `internal/core/errors_test.go` — add test for HookError.Error() string format
- [x] 1.8 GREEN: `internal/core/errors.go` — add HookError type (event name + wrapped error)
- [x] 1.9 Verify: `go test ./internal/core/ -run "TestExtension|TestHookRegistry|TestHookContext|TestHookError"` — all pass

## Phase 2: Loop Integration (internal/core/loop.go)

- [ ] 2.1 RED: `internal/core/loop_hook_test.go` — test nil HookRegistry = no-op (existing behavior), non-nil fires hooks, before_provider_request mutates messages, before_tool_execution blocks tool, after_tool_execution observes result, hook error logged but doesn't abort loop
- [ ] 2.2 GREEN: `internal/core/loop.go` — add `Hooks *HookRegistry` field to Agent struct; emit `before_provider_request` before Chat/StreamChat; emit `before_tool_execution` + block check before tool.Execute; emit `after_tool_execution` after result; wrap emit calls in recover (D8)
- [ ] 2.3 GREEN: `internal/core/loop.go` — add `emitHook(registry, event, ctx)` helper with per-handler recover (mirrors emitObserver pattern)
- [ ] 2.4 Verify: `go test ./internal/core/ -run "TestLoopHook|TestLoop"` — all pass; `go test ./internal/core/ -run TestCoreImportsStdlibOnly` — guard still green
- [ ] 2.5 Verify: `go test ./internal/core/...` — full suite passes (backward compatibility)

## Phase 3: Discovery + Example (internal/adapters/extensions)

- [ ] 3.1 RED: `internal/adapters/extensions/registry_test.go` — test Register appends, Register(nil) panics, LoadAll calls Init in order, LoadAll rolls back on failure, ShutdownAll reverse order, ShutdownAll idempotent, ShutdownAll collects errors
- [ ] 3.2 GREEN: `internal/adapters/extensions/registry.go` — package-level `Register(ext)`, `LoadAll(api ExtensionAPI)`, `ShutdownAll()` functions; registration slice, init-order forward / reverse-order shutdown, error collection
- [ ] 3.3 RED: `internal/adapters/extensions/example_ext_test.go` — test example extension Init/Shutdown lifecycle, tool registration, hook registration
- [ ] 3.4 GREEN: `internal/extensions/example/example.go` — example extension implementing Extension interface; registers a tool and a hook during Init
- [ ] 3.5 GREEN: `cmd/kui/main.go` — add blank import `_ "github.com/biggs-100/kui/internal/extensions/example"` (opt-in registration)
- [ ] 3.6 Verify: `go test ./internal/adapters/extensions/...` — all pass
- [ ] 3.7 Verify: `go build ./cmd/kui` — compiles with example extension imported
- [ ] 3.8 Verify: `go test ./...` — full project suite passes
