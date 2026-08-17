# Delta for extension-wiring

Targets main spec: `openspec/specs/extension-system/spec.md`. Existing requirements REQ-EXT-1..6 remain valid unchanged; this delta adds production-wiring obligations that did not previously exist.

## ADDED Requirements

### Requirement: REQ-RELOAD-16 — Concrete ExtensionAPI in Production

The production runtime MUST provide a concrete `ExtensionAPI` implementation backed by the tool registry and the hook registry, available before any `LoadAll` call, so extensions can register tools, hooks, and commands during `Init`.

#### Scenario: Extension registers through the production API

- GIVEN the production runtime build
- WHEN LoadAll is called with the concrete API
- THEN a registering extension's tool appears in the tool registry
- AND its hooks appear in the hook registry

#### Scenario: Duplicate registration still errors

- GIVEN two extensions registering the same tool name
- WHEN the concrete API processes the second registration
- THEN the duplicate-name error is returned
- AND the first tool remains registered

### Requirement: REQ-RELOAD-17 — LoadAll in Production Build

The production runtime build MUST call `extensions.LoadAll(api)` so compiled-in extensions become active. A LoadAll error MUST fail the build — and on reload, build-new-then-swap MUST keep the previous runtime active.

#### Scenario: Startup loads extensions

- GIVEN a runtime build
- WHEN Build runs
- THEN LoadAll is called with the concrete API
- AND active extensions are initialized

#### Scenario: LoadAll failure fails the build

- GIVEN an extension whose Init fails
- WHEN Build runs
- THEN Build returns the error
- AND on reload the previous runtime stays active

### Requirement: REQ-RELOAD-18 — ShutdownAll on Reload and Close

The runtime MUST call `extensions.ShutdownAll()` during teardown — on reload before rebuild, and on Close.

#### Scenario: Reload tears down extensions

- GIVEN active extensions
- WHEN Reload runs
- THEN ShutdownAll is called before the rebuild
- AND the new build's LoadAll re-initializes extensions

#### Scenario: Close shuts down extensions once

- GIVEN active extensions
- WHEN Close is called
- THEN ShutdownAll runs once
- AND a second Close does not re-run it