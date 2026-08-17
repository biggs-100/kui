# runtime-lifecycle Specification

## Purpose

`internal/runtime` owns kui's runtime composition. `Build` assembles a complete runtime snapshot (provider, profiles, skills, MCP, hooks, steering); `Reload` cancels-and-waits active runs, tears down MCP and extensions, re-reads configurable state, and swaps only on a clean build; `Close` releases resources. It replaces the duplicated composition in `runPrompt` and `tui.Run`.

## Requirements

### Requirement: REQ-RELOAD-1 — Runtime Build

The system MUST provide `internal/runtime` with `Build(ctx, cfg)` that composes all runtime components: provider, profiles loader, skills index, MCP manager, tool registry, hooks, and steering/follow-up queues. Build MUST return a self-contained runtime snapshot.

#### Scenario: Build composes a full snapshot

- GIVEN a valid runtime config
- WHEN Build is called
- THEN the snapshot contains a working provider, profile manager, skills index, connected MCP manager, tool registry, and hooks

#### Scenario: Build propagates composition errors

- GIVEN an invalid provider configuration
- WHEN Build is called
- THEN Build returns the error
- AND no partially-built runtime is returned

### Requirement: REQ-RELOAD-2 — Single Build Path

Build MUST be the single composition path: both `runPrompt` in `cmd/kui` and `tui.Run` MUST obtain their runtime from `runtime.Build` instead of composing components inline.

#### Scenario: runPrompt delegates to Build

- GIVEN the CLI entrypoint
- WHEN runPrompt starts
- THEN it obtains its runtime from runtime.Build

#### Scenario: tui.Run delegates to Build

- GIVEN the TUI entrypoint
- WHEN tui.Run starts
- THEN it obtains its runtime from runtime.Build
- AND no inline component composition remains in either entrypoint

### Requirement: REQ-RELOAD-3 — Reload Re-reads State

`Reload(ctx)` MUST re-read all configurable state: profiles, skills, MCP config, provider, hooks, and steering re-seed. Rebuilt components MUST reflect on-disk state at reload time.

#### Scenario: New skill and changed profile picked up

- GIVEN a skill file and a profile added since startup
- WHEN Reload runs
- THEN the new skill is indexed
- AND the changed profile is discovered and resolvable

#### Scenario: MCP and provider rebuilt

- GIVEN an mcp.yaml with a new server
- WHEN Reload runs
- THEN the new MCP server connects
- AND the provider is recreated from current config

### Requirement: REQ-RELOAD-4 — Build-New-Then-Swap

Reload MUST build the new runtime before swapping. On failure, the old runtime MUST remain fully active and the error MUST be surfaced; a failed reload MUST NOT leave a partial runtime active.

#### Scenario: Successful swap

- GIVEN a clean rebuild
- WHEN Reload completes
- THEN the new runtime is active
- AND old components are shut down

#### Scenario: Failed build keeps old state

- GIVEN a rebuild that fails (e.g. skills index error)
- WHEN Reload runs
- THEN the error is returned
- AND the previous runtime remains fully active and usable

### Requirement: REQ-RELOAD-5 — Close Teardown

`Close()` MUST shut down the MCP manager and extensions cleanly and MUST be idempotent.

#### Scenario: Clean close

- GIVEN a running runtime with connected MCP and active extensions
- WHEN Close is called
- THEN all MCP servers shut down
- AND extensions.ShutdownAll runs
- AND no goroutines leak

#### Scenario: Close is idempotent

- GIVEN a closed runtime
- WHEN Close is called again
- THEN it returns without error
- AND no resources are shut down twice