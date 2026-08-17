# hook-registry Specification

## Purpose

HookRegistry manages event-to-handler mappings, emitting hooks in registration order with error short-circuit semantics.

## Requirements

### Requirement: REQ-HOOK-1 — Register Method

`HookRegistry` MUST expose `Register(event string, handler HookHandler) error`. Registration MUST be deterministic — handlers execute in the order they were registered. Registering a nil handler MUST return an error.

#### Scenario: Register handler for known event

- GIVEN a HookRegistry instance
- WHEN Register("on_turn_start", handler) is called
- THEN the handler is stored and associated with "on_turn_start"

#### Scenario: Register nil handler

- GIVEN a HookRegistry instance
- WHEN Register("on_turn_start", nil) is called
- THEN Register returns an error
- AND no handler is stored

### Requirement: REQ-HOOK-2 — Emit in Registration Order

`HookRegistry` MUST expose `Emit(event string, ctx HookContext) error`. Emit MUST call all handlers registered for the event in the order they were registered. Emit MUST return nil if no handlers are registered.

#### Scenario: Multiple handlers called in order

- GIVEN handlers H1, H2, H3 registered for "on_tool_call"
- WHEN Emit("on_tool_call", ctx) is called
- Then H1 runs first, H2 second, H3 third
- AND Emit returns nil

#### Scenario: Emit with no handlers

- GIVEN no handlers registered for "on_system_prompt"
- WHEN Emit("on_system_prompt", ctx) is called
- Then Emit returns nil without error

### Requirement: REQ-HOOK-3 — Error Short-Circuit

When any handler returns a non-nil error, Emit MUST stop executing remaining handlers and return that error immediately. The error MUST include the event name and the handler's error.

#### Scenario: First handler errors — chain stops

- GIVEN H1 (returns nil) and H2 (returns errX) for "on_tool_result"
- WHEN Emit is called
- Then H1 executes successfully
- AND H2 executes and returns errX
- AND no further handlers run
- AND Emit returns errX

#### Scenario: First handler errors — second never runs

- GIVEN H1 (returns errY) and H2 (never called) for "on_turn_end"
- WHEN Emit is called
- Then H1 executes and returns errY
- AND H2 is never invoked

### Requirement: REQ-HOOK-4 — HasHooks Fast-Path

`HookRegistry` MUST expose `HasHooks(event string) bool` that returns true only if at least one handler is registered for the event. This enables callers to skip HookContext allocation when no hooks exist.

#### Scenario: HasHooks returns true for registered event

- GIVEN a handler registered for "on_tool_call"
- WHEN HasHooks("on_tool_call") is called
- Then it returns true

#### Scenario: HasHooks returns false for unregistered event

- GIVEN no handlers for "on_session_end"
- WHEN HasHooks("on_session_end") is called
- Then it returns false

### Requirement: REQ-HOOK-5 — Nil-Safe Registry

A nil `*HookRegistry` field on any struct MUST behave identically to an empty registry: Emit returns nil, HasHooks returns false, Register panics (nil pointer). This ensures backward compatibility — existing code with nil HookRegistry fields compiles and runs without changes.

#### Scenario: Nil registry Emit is no-op

- GIVEN a struct with a nil *HookRegistry field
- WHEN Emit is called on the nil pointer
- Then no panic occurs (method has nil receiver)
- AND Emit returns nil

#### Scenario: Nil registry HasHooks returns false

- GIVEN a struct with a nil *HookRegistry field
- WHEN HasHooks is called
- Then it returns false
