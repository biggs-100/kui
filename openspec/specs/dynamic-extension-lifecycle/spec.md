# Dynamic Extension Lifecycle

## Purpose

Subprocess spawning, JSON-RPC protocol, tool registration, crash handling, and shutdown for runtime extensions.

## Requirements

### Requirement: Subprocess Spawning

The system MUST spawn extensions as subprocesses via JSON-RPC 2.0 stdio. Spawn failures MUST log and mark extension unavailable.

#### Scenario: Entry point missing

- GIVEN missing entry point
- WHEN spawn attempted
- THEN extension unavailable; others continue

### Requirement: Protocol Handshake

The system MUST send `initialize` with protocol version. Mismatch MUST reject with clear error.

#### Scenario: Version mismatch rejected

- GIVEN incompatible protocol version
- WHEN `initialize` sent
- THEN extension terminated

#### Scenario: Handshake succeeds

- GIVEN valid extension
- WHEN `initialize` sent
- THEN capabilities returned and tools discovered

### Requirement: Tool Registration

Tools MUST use `core.Tool` interface, prefixed `{extensionName}_` to avoid collisions.

#### Scenario: Prefixed tools

- GIVEN extension `notes` with tool `read`
- WHEN registered
- THEN available as `notes_read`

### Requirement: Crash Handling

Extension crash MUST be non-fatal. Tools marked unavailable. User recovers via `/reload`.

#### Scenario: Extension crashes

- GIVEN active extension
- WHEN subprocess exits
- THEN other extensions unaffected; tools return error

### Requirement: Graceful Shutdown

On reload/exit, system MUST send `shutdown`, wait, then kill after timeout.

#### Scenario: Clean shutdown

- GIVEN active extensions
- WHEN shutdown triggered
- THEN extensions exit cleanly

#### Scenario: Timeout kill

- GIVEN extension ignoring shutdown
- WHEN timeout expires
- THEN subprocess killed
