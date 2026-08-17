# extension-system Specification

## Purpose

Extension system providing lifecycle interfaces, hook registry, and mutable hook context for extensions that modify agent behavior without touching core files.

## Requirements

### Requirement: REQ-EXT-1 — Extension Interface

The system MUST define an `Extension` interface with three methods: `Name() string`, `Init(ExtensionAPI) error`, and `Shutdown() error`. `Name` returns a stable identifier. `Init` receives the extension API and MUST complete successfully for the extension to become active. `Shutdown` MUST release all resources.

#### Scenario: Extension initializes successfully

- GIVEN an extension implementing the Extension interface
- WHEN Init is called with a valid ExtensionAPI
- THEN the extension returns nil and is ready to serve hooks/tools/commands

#### Scenario: Init returns an error

- GIVEN an extension whose Init fails (e.g., missing config)
- WHEN LoadAll calls Init
- THEN the extension is NOT registered as active
- AND LoadAll returns the error without calling Shutdown on that extension

### Requirement: REQ-EXT-2 — ExtensionAPI Interface

The system MUST define an `ExtensionAPI` interface exposing `RegisterTool(Tool) error`, `RegisterHook(event string, handler HookHandler) error`, and `RegisterCommand(Command) error`. All registration methods MUST be safe to call during `Init`.

#### Scenario: Extension registers a tool during Init

- GIVEN an extension with a custom tool
- WHEN Init calls api.RegisterTool(tool)
- THEN the tool becomes available in the agent's tool registry

#### Scenario: RegisterTool with duplicate name

- GIVEN two extensions registering tools with the same name
- WHEN the second RegisterTool call occurs
- THEN RegisterTool returns a duplicate-name error
- AND the first tool remains registered

### Requirement: REQ-EXT-3 — HookHandler Type

The system MUST define `HookHandler` as `func(HookContext) error`. Handlers MUST be invoked by the HookRegistry in registration order. A handler returning a non-nil error MUST short-circuit the remaining handlers for that event.

#### Scenario: Handler executes in registration order

- GIVEN two handlers registered for event "on_turn_start"
- WHEN the hook is emitted
- THEN the first-registered handler runs before the second

#### Scenario: Handler error stops chain

- GIVEN handler A (returns nil) and handler B (returns error) for the same event
- WHEN the hook is emitted
- THEN handler A runs successfully
- AND handler B runs and returns error
- AND no subsequent handlers execute

### Requirement: REQ-EXT-4 — HookContext Interface

The system MUST define a `HookContext` interface exposing `Messages() []Message`, `SetMessages([]Message)`, `ToolCall() *ToolCall`, `SetToolCall(*ToolCall)`, `Block(reason string)`, `IsBlocked() bool`, and `BlockReason() string`. `Block()` MUST prevent downstream actions (e.g., tool execution) and store the reason.

#### Scenario: Handler modifies messages

- GIVEN a HookContext with 3 messages
- WHEN a handler calls SetMessages with 2 messages
- THEN subsequent handlers and the loop see the modified 2-message set

#### Scenario: Handler blocks tool execution

- GIVEN a HookContext during a before_tool_execution hook
- WHEN a handler calls Block("security policy")
- THEN IsBlocked() returns true
- AND BlockReason() returns "security policy"
- AND the tool is NOT executed

#### Scenario: Messages returns nil-safe slice

- GIVEN a HookContext with no messages set
- WHEN Messages() is called
- THEN it returns an empty or nil slice without panicking

### Requirement: REQ-EXT-5 — Compiled-In Registration

Extensions MUST be registered via Go `init()` patterns in their own packages. The `extensions.Register(ext)` function MUST add the extension to a package-level slice. Registration MUST be deterministic and happen before any `LoadAll` call.

#### Scenario: Extension self-registers via init

- GIVEN a package with an init() function calling extensions.Register(myExt)
- WHEN the binary imports the package
- THEN myExt appears in the registered extensions list

#### Scenario: No extensions registered

- GIVEN no packages with extension init() are imported
- WHEN LoadAll is called
- THEN it returns successfully with zero extensions loaded

### Requirement: REQ-EXT-6 — Extension Lifecycle

Extensions MUST follow the lifecycle: discover → Init (in registration order) → active → Shutdown (in reverse registration order). If any Init fails, previously-initialized extensions MUST receive Shutdown in reverse order and LoadAll MUST return the error.

#### Scenario: Normal lifecycle with three extensions

- GIVEN extensions A, B, C registered in that order
- WHEN LoadAll completes
- Then A.Init, B.Init, C.Init were called in order
- AND all three are active

#### Scenario: B.Init fails — rollback A and C

- GIVEN extensions A, B, C where B.Init returns an error
- WHEN LoadAll processes them
- Then A.Init succeeded
- AND B.Init failed
- AND A.Shutdown is called (reverse order of successful inits)
- AND LoadAll returns B's error
