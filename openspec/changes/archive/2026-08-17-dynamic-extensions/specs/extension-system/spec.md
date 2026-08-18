# Extension System — Delta

## MODIFIED Requirements

### Requirement: REQ-EXT-5 — Compiled-In Registration

Extensions MUST register via Go `init()` using `extensions.Register(ext)`. The system MUST also support `extensions.RegisterDynamic(ext)` for filesystem-discovered extensions. Both MUST complete before `LoadAll`.
(Previously: Compiled-in registration only)

#### Scenario: Dynamic extension registered

- GIVEN a dynamically discovered extension
- WHEN RegisterDynamic(ext) called before LoadAll
- THEN included in LoadAll

### Requirement: REQ-EXT-6 — Extension Lifecycle

Extensions follow: discover → Init (order) → active → Shutdown (reverse). Init failure triggers reverse Shutdown. Dynamic extensions follow same lifecycle.
(Previously: Compiled-in only)

#### Scenario: Mixed extensions coexist

- GIVEN compiled-in A and dynamic B
- WHEN LoadAll processes both
- THEN both active simultaneously

#### Scenario: Init failure with mixed extensions

- GIVEN compiled-in A and dynamic B where B.Init fails
- WHEN LoadAll processes them
- THEN A.Shutdown called; error returned
