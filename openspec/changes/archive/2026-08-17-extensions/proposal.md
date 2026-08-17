# Proposal: Extension System with Lifecycle Hooks

## Intent

kui lacks extensibility — every tool and behavior is hardcoded in `internal/adapters/tools/` and `internal/core/loop.go`. There's no way to inject domain logic, add custom tools, or modify agent behavior without touching core files. This blocks community contributions, prevents domain-specific adapters (code review hooks, custom MCP bridges), and creates parity debt with pi/opencode which have extension systems.

## Scope

### In Scope
- `Extension` interface — Init/Shutdown lifecycle, registered via Go `init()` pattern
- `HookRegistry` — 8 lifecycle hooks with mutable `HookContext`
- Hook integration into `core.Agent.Run()` at defined lifecycle points
- Extension discovery via `init()` auto-registration
- Example extension demonstrating the pattern

### Out of Scope
- Runtime/dynamic extension loading (WASM, plugin DLLs)
- Hot-reload or extension hot-swap
- Extension marketplace or versioning
- Extension configuration schema (deferred to profile system)

## Capabilities

### New Capabilities
- `extension-system`: Extension interface, HookRegistry, HookContext, lifecycle hooks, discovery mechanism

### Modified Capabilities
- `agent-loop`: HookRegistry wired into loop lifecycle points (turn start/end, tool call/result, system prompt assembly)
- `agent-tools`: Extensions register tools through HookRegistry; tool discovery includes extension-contributed tools

## Approach

Hexagonal: core defines `Extension` interface and `HookRegistry` port. Adapters implement discovery via Go `init()`. Hooks are typed functions receiving mutable `HookContext` — extensions can modify messages, block tools, or replace the system prompt. Observer stays read-only for TUI; hooks are the mutable counterpart.

Compiled-in: extensions are Go packages imported in `cmd/kui/main.go`. No runtime loading — keeps hexagonal guard, stdlib-only core, and binary simplicity.

8 initial hooks: OnSessionStart, OnSessionEnd, OnTurnStart, OnTurnEnd, OnToolCall, OnToolResult, OnSystemPrompt, OnProviderRequest. Each receives `HookContext` with conversation history, active profile, and tool registry reference.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/core/extension.go` | New | Extension interface, HookContext, HookRegistry port |
| `internal/core/loop.go` | Modified | Wire HookRegistry at 8 lifecycle points |
| `internal/adapters/extensions/` | New | Discovery adapter (init() registration) |
| `internal/adapters/tools/registry.go` | Modified | Accept extension-contributed tools |
| `cmd/kui/main.go` | Modified | Import example extension package |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Hook execution order nondeterminism | Med | Registration order = execution order; document contract |
| Hook panics crash loop | Low | Wrap in recover (same pattern as Observer) |
| Core guard test breaks | Low | HookRegistry is a port (interface) in core; adapter lives outside |

## Rollback Plan

Remove `extension.go`, revert `loop.go` hook calls, delete `adapters/extensions/`. The loop reverts to current behavior with zero hooks. No data migration needed — extensions are stateless.

## Dependencies

- None (pure Go, stdlib only in core)

## Success Criteria

- [ ] `Extension` interface compiles with Init/Shutdown
- [ ] `HookRegistry` fires all 8 hooks at correct loop points
- [ ] Example extension registers a tool and modifies system prompt
- [ ] `TestCoreImportsStdlibOnly` guard test still passes
- [ ] All existing loop tests pass unmodified
