# profile-skills Specification

## Purpose

Skills are discoverable units — name, description, triggers, and a SKILL.md body — found in layered directories (global → project → profile, nearest wins). The system prompt carries only the skills index; full bodies load on demand at invocation.

## Requirements

### Requirement: REQ-SKILL-1 — Layered Discovery

Skills MUST be discovered across the global, project, and profile layers. When the same skill name exists in multiple layers, the nearest layer (profile > project > global) MUST win.

#### Scenario: Collision resolution

- GIVEN skill "go-testing" in both the global and profile layers
- WHEN the index is built
- THEN exactly one entry exists, sourced from the profile layer

#### Scenario: Layered aggregation

- GIVEN distinct skills in each of the three layers
- WHEN the index is built
- THEN all skills are present

### Requirement: REQ-SKILL-2 — Index and Trigger Matching

Each skill MUST expose a name, description, and triggers. A skill MUST be applicable when a message matches at least one trigger. The index MUST be buildable without reading any skill body.

#### Scenario: Trigger match

- GIVEN a skill with trigger "go test"
- WHEN a message containing "go test" is matched
- THEN the skill is applicable

#### Scenario: No trigger match

- GIVEN a message unrelated to any trigger
- WHEN matching runs
- THEN no skills are applicable

### Requirement: REQ-SKILL-3 — On-Demand Content

The system prompt MUST contain only the skills index (names, descriptions, triggers) and MUST NOT contain any skill body. The full SKILL.md content MUST be loaded only when the skill is invoked.

#### Scenario: Index-only prompt

- GIVEN three skills with bodies
- WHEN the system prompt is built
- THEN the prompt lists names and triggers
- AND contains no body text

#### Scenario: Body loads on invocation

- GIVEN a skill selected for invocation
- WHEN its content is requested
- THEN the full SKILL.md content is loaded at that moment

#### Scenario: Missing body file

- GIVEN an index entry whose SKILL.md does not exist
- WHEN the skill is invoked
- THEN a typed error naming the missing file is returned
