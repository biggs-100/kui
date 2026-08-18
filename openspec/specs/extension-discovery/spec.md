# extension-discovery Specification

## Purpose

Extension discovery manages the compiled-in registration of extensions, their initialization, and ordered shutdown.

## Requirements

### Requirement: REQ-DISCOVERY-1 — Startup Discovery

Extensions MUST be discovered via compiled-in registry AND filesystem scanning. Discovery MUST complete before `LoadAll` call. The compiled-in registry is populated by `init()` functions; filesystem scanning checks global (`~/.config/kui/extensions/`) and project-level (`.kui/extensions/`) directories. Project extensions MUST override global on name collision.

#### Scenario: Extensions discovered from imported packages

- GIVEN packages with init() functions that call extensions.Register()
- WHEN the binary starts and imports those packages
- Then the registered extensions are available for LoadAll

#### Scenario: No extensions imported

- GIVEN no extension packages imported in main.go
- When discovery completes
- Then the registry is empty
- AND LoadAll succeeds with zero extensions

#### Scenario: Filesystem extensions discovered

- GIVEN extensions in global and project directories
- WHEN discovery completes
- THEN added to registry; project overrides global

### Requirement: REQ-DISCOVERY-2 — Register Function

`extensions.Register(ext Extension)` MUST append the extension to a package-level slice. `extensions.RegisterDynamic(ext)` MUST follow the same contract for filesystem-discovered extensions. Register MUST NOT be called after LoadAll has started. Calling Register with a nil extension MUST panic or return an error (fail-fast).

#### Scenario: Register adds extension to list

- GIVEN a valid extension instance
- WHEN extensions.Register(ext) is called
- Then ext appears in the registered list

#### Scenario: Register nil panics

- GIVEN a nil Extension value
- When extensions.Register(nil) is called
- Then the call panics with a clear message

#### Scenario: RegisterDynamic adds runtime extension

- GIVEN valid dynamic extension
- WHEN RegisterDynamic(ext) called before LoadAll
- THEN ext in list alongside compiled-in

### Requirement: REQ-DISCOVERY-3 — LoadAll Initialization

`extensions.LoadAll(api ExtensionAPI)` MUST initialize all registered extensions in registration order by calling Init(api) on each. If Init returns an error, LoadAll MUST stop, call Shutdown on all previously-initialized extensions in reverse order, and return the error.

#### Scenario: All extensions initialize successfully

- GIVEN 3 registered extensions A, B, C
- When LoadAll(api) is called
- Then A.Init, B.Init, C.Init are called in order
- AND LoadAll returns nil

#### Scenario: Middle extension fails — rollback

- GIVEN 3 registered extensions where B.Init returns errorX
- When LoadAll(api) is called
- Then A.Init succeeds
- AND B.Init fails
- AND A.Shutdown is called (reverse of successful inits)
- AND LoadAll returns errorX

### Requirement: REQ-DISCOVERY-4 — ShutdownAll Cleanup

`extensions.ShutdownAll()` MUST call Shutdown on all active extensions in reverse registration order. ShutdownAll MUST be idempotent — calling it twice does not double-shutdown. Errors during Shutdown MUST be collected and returned (not short-circuited).

#### Scenario: Normal shutdown in reverse order

- GIVEN 3 active extensions A, B, C (registered in that order)
- When ShutdownAll is called
- Then C.Shutdown, B.Shutdown, A.Shutdown are called in order
- AND ShutdownAll returns nil

#### Scenario: Shutdown error collected, not short-circuited

- GIVEN active extensions A, B, C where B.Shutdown returns errZ
- When ShutdownAll is called
- Then C.Shutdown runs successfully
- AND B.Shutdown returns errZ
- AND A.Shutdown runs successfully
- AND ShutdownAll returns errZ (or a combined error)

#### Scenario: Idempotent shutdown

- GIVEN ShutdownAll called once successfully
- When ShutdownAll is called again
- Then no Shutdown methods are called
- AND ShutdownAll returns nil
