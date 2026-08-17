# Design: Extension System with Lifecycle Hooks

## Technical Approach

Add an `Extension` interface, `HookRegistry`, and `HookContext` to `internal/core/` (stdlib-only, preserving the hexagonal guard). Extensions register via Go `init()` self-registration in `internal/adapters/extensions/`. The loop gains a nil-safe `*HookRegistry` field on `core.Agent` — when non-nil, hooks fire at three lifecycle points with mutable context. Observer stays read-only for TUI; hooks are the mutable counterpart.

## Architecture Decisions

| # | Decision | Choice | Alternatives | Rationale |
|---|----------|--------|-------------|-----------|
| D1 | Extension interface | `Name()`, `Init(ExtensionAPI) error`, `Shutdown() error` | Func-based, struct-based | Interface enables compile-time check; Init/Shutdown mirror Go module lifecycle; matches hexagonal port pattern |
| D2 | ExtensionAPI scope | `RegisterTool`, `RegisterHook`, `RegisterCommand` | Single `Register` with variadic args | Typed methods catch misuse at compile time; `RegisterCommand` stubbed for future TUI commands |
| D3 | HookRegistry structure | `map[string][]HookHandler` with registration-order slice | `sync.Map`, sorted tree | Registration order = execution order (deterministic); map lookup is O(1); no concurrency needed (single-threaded loop) |
| D4 | HookContext mutability | Mutable struct with `Messages()`/`SetMessages()`, `Block()`/`IsBlocked()` | Immutable context + return values | Matches Observer pattern (mutable callback); `Block()` is cleaner than returning flags; matches proposal REQ-EXT-4 |
| D5 | Hook injection points | 3 hooks: `before_provider_request`, `before_tool_execution`, `after_tool_execution` | 8 hooks per proposal | Start minimal — 3 hooks cover message modification, tool gating, and result observation; add session/turn hooks later if needed |
| D6 | Discovery mechanism | Compiled-in via `init()` + `extensions.Register(ext)` | Dynamic loading (WASM, plugin DLLs) | Keeps hexagonal guard; stdlib-only core; binary simplicity; matches proposal scope |
| D7 | Backward compatibility | Nil-safe `*HookRegistry` on `Agent` struct | Separate loop variant, feature flag | Zero-cost when unused; nil receiver methods return no-op; existing tests pass unchanged |
| D8 | Panic recovery | Per-handler `recover()` wrapping (same as Observer pattern) | Global defer | Matches `emitObserver` pattern; isolates panics per handler; loop never crashes |

## Data Flow

```
extensions.LoadAll(api)          ← startup, before loop
  │
  ▼
Agent.Run(prompt)
  │
  ├─→ [before_provider_request]  ← mutate messages before LLM call
  │     HookContext{messages}  →  handlers may call SetMessages()
  │
  ▼
Provider.Chat() / StreamChat()
  │
  ▼
for each ToolCall in response:
  │
  ├─→ [before_tool_execution]    ← gate tool via Block()
  │     HookContext{toolCall}  →  handlers may call Block(reason)
  │     if blocked: skip execution, return blocked result
  │
  ▼
tool.Execute()
  │
  ├─→ [after_tool_execution]     ← observe result (read-only)
  │     HookContext{toolCall, result}
  │
  ▼
loop continues
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/core/extension.go` | Create | `Extension` interface, `ExtensionAPI` interface, `HookHandler` type, `HookContext` interface |
| `internal/core/hook_registry.go` | Create | `HookRegistry` struct: `Register`, `Emit`, `HasHooks` methods; nil-safe pointer receiver |
| `internal/core/hook_context.go` | Create | Concrete `hookContext` struct implementing `HookContext`; mutable message/tool/block state |
| `internal/core/loop.go` | Modify | Add `HookRegistry *HookRegistry` field to `Agent`; emit hooks at 3 lifecycle points |
| `internal/core/errors.go` | Modify | Add `HookError` type (event name + wrapped error) for short-circuit propagation |
| `internal/adapters/extensions/registry.go` | Create | Package-level `Register(ext)`, `LoadAll(api)`, `ShutdownAll()` functions |
| `internal/adapters/extensions/example_test.go` | Create | Test extension demonstrating Init/Shutdown/RegisterTool/RegisterHook |
| `cmd/kui/main.go` | Modify | Import `_ "github.com/biggs-100/kui/internal/adapters/extensions"` (blank import for init) |

## Interfaces / Contracts

```go
// internal/core/extension.go
type Extension interface {
    Name() string
    Init(api ExtensionAPI) error
    Shutdown() error
}

type ExtensionAPI interface {
    RegisterTool(tool Tool) error
    RegisterHook(event string, handler HookHandler) error
    RegisterCommand(cmd Command) error  // stubbed for future
}

type HookHandler func(HookContext) error

type HookContext interface {
    Messages() []Message
    SetMessages([]Message)
    ToolCall() *ToolCall
    SetToolCall(*ToolCall)
    Block(reason string)
    IsBlocked() bool
    BlockReason() string
}
```

```go
// internal/core/hook_registry.go
type HookRegistry struct {
    handlers map[string][]HookHandler
}

func (r *HookRegistry) Register(event string, handler HookHandler) error
func (r *HookRegistry) Emit(event string, ctx HookContext) error  // nil receiver = no-op
func (r *HookRegistry) HasHooks(event string) bool               // nil receiver = false
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | HookRegistry: register order, emit order, error short-circuit, HasHooks, nil-safety | Table-driven tests with mock handlers |
| Unit | HookContext: message mutation, block/unblock, nil-safe messages | Direct construction, assert state changes |
| Unit | Extension lifecycle: init order, shutdown rollback on failure, idempotent shutdown | Mock extensions with controlled Init/Shutdown |
| Integration | Loop with HookRegistry: hooks fire at correct points, blocked tools skip execution | Wire HookRegistry into `core.Agent`, mock provider+tools |
| Integration | Loop without HookRegistry: all existing tests pass unchanged | Verify nil field = no-op behavior |
| Guard | `TestCoreImportsStdlibOnly` still passes | Core package must not import adapters |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. All changes are additive:
- `HookRegistry` field is nil by default; existing `Agent` construction unaffected
- Observer stays for TUI rendering; hooks are separate concern
- Extensions are opt-in via blank import in `cmd/kui/main.go`
- `TestCoreImportsStdlibOnly` guard test remains green (extension.go + hook_registry.go are in core, stdlib-only)

## Open Questions

- [ ] Should `RegisterCommand` be included now (stubbed) or deferred to when TUI commands are needed? Design includes stub per proposal scope.
