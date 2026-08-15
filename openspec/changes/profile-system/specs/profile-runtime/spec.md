# profile-runtime Specification

## Purpose

Profiles are first-class runtime units: each profile.yaml declares a name, model, system prompt reference, tools, skills, and permissions. A layered resolver (global → project → profile) locates configuration, a manager holds the active profile, and hot switches apply between turns within the same session and history. Per-profile model choices persist in `.kui/`.

## Requirements

### Requirement: REQ-PROFILE-1 — Profile Model

A profile MUST declare name, model, system_prompt (a SYSTEM.md path), tools, skills, and permissions. The loader MUST parse profile.yaml through the loader port and MUST return a typed error naming the offending file on malformed or missing content.

#### Scenario: Valid profile

- GIVEN profile.yaml with name "coder", model "gpt-4o", and a SYSTEM.md path
- WHEN the loader parses it
- THEN a profile with those values is produced

#### Scenario: Malformed yaml

- GIVEN profile.yaml with invalid YAML syntax
- WHEN the loader parses it
- THEN a typed parse error naming the file is returned

#### Scenario: Missing SYSTEM.md

- GIVEN a profile whose system_prompt references a file that does not exist
- WHEN the profile is activated
- THEN activation fails with a typed error naming the missing file

### Requirement: REQ-PROFILE-2 — Layered Resolution

Configuration MUST resolve from three layers in order — global, project, profile — with the nearest layer winning per field. Resolution MUST be a pure function with no I/O.

#### Scenario: Override precedence

- GIVEN a model in global config and a different model in the project layer
- WHEN resolution runs for a profile declaring no model
- THEN the project-layer model wins

#### Scenario: Empty profile layer

- GIVEN a profile declaring no model
- WHEN resolution runs
- THEN the fallback is the project layer, then the global default

### Requirement: REQ-PROFILE-3 — Hot Switch Between Turns

A profile switch MUST be queued and MUST apply only between turns — never while a tool executes and never mid-response. The conversation history MUST be preserved unchanged across the switch.

#### Scenario: Switch while tool runs

- GIVEN a switch request queued during a tool call
- WHEN the tool call completes
- THEN the switch applies before the next provider turn

#### Scenario: History preserved

- GIVEN an active conversation with prior messages
- WHEN a profile switch applies
- THEN all prior messages remain in the history

#### Scenario: Unknown profile

- GIVEN a switch request naming a profile that does not exist
- WHEN the switch is processed
- THEN a typed unknown-profile error is returned
- AND the active profile is unchanged

### Requirement: REQ-PROFILE-4 — Model Memory

The runtime MUST persist the active model per profile through the model-memory port and MUST restore it when that profile is activated in a later session. Persistence MUST live in the `.kui/` adapter, never in the core.

#### Scenario: Persist and restore

- GIVEN profile "coder" switched to model "gpt-4o" in session one
- WHEN "coder" is activated in session two
- THEN the model resolves to "gpt-4o"

#### Scenario: No saved model

- GIVEN no saved model for a profile
- WHEN it is activated
- THEN resolution falls back to the layered config

### Requirement: REQ-PROFILE-5 — Adapter Boundary

The core MUST NOT import YAML-parsing or filesystem packages: profile loading, resolution persistence, and model memory MUST be reached only through ports implemented in adapters. The hexagonal guard test MUST enforce this boundary.

#### Scenario: Guard enforced

- GIVEN the core package compiled against the guard test
- WHEN the guard inspects imports
- THEN no yaml or filesystem dependency appears in the core

#### Scenario: Adapter implements port

- GIVEN the loader adapter
- WHEN the core calls the loader port
- THEN the adapter performs all yaml and filesystem work
