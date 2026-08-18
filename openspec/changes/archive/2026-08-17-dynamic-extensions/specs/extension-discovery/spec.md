# Extension Discovery — Delta

## MODIFIED Requirements

### Requirement: REQ-DISCOVERY-1 — Startup Discovery

Extensions MUST be discovered via compiled-in registry AND filesystem scanning. Discovery MUST complete before `LoadAll`.
(Previously: Compiled-in registry only)

#### Scenario: Filesystem extensions discovered

- GIVEN extensions in global and project directories
- WHEN discovery completes
- THEN added to registry; project overrides global

### Requirement: REQ-DISCOVERY-2 — Register Function

`extensions.Register(ext)` appends to package-level slice. `extensions.RegisterDynamic(ext)` follows same contract. Nil MUST panic.
(Previously: Register() only)

#### Scenario: RegisterDynamic adds runtime extension

- GIVEN valid dynamic extension
- WHEN RegisterDynamic(ext) called before LoadAll
- THEN ext in list alongside compiled-in
