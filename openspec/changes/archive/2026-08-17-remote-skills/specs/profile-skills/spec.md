# profile-skills Delta — MODIFIED by remote-skills

## Purpose

Skills are discoverable units — name, description, triggers, and a SKILL.md body — found in layered directories. This delta extends the layer model with remote skills fetched from registries, and allows the `skills` config field to accept URLs alongside local names.

## MODIFIED Requirements

### Requirement: REQ-SKILL-1 — Layered Discovery

Skills MUST be discovered across the global, remote, project, and profile layers. When the same skill name exists in multiple layers, the nearest layer (profile > project > remote > global) MUST win. Remote skills are fetched from registry URLs configured in the profile `skills` field.
(Previously: Three-layer discovery — global → project → profile, nearest wins.)

#### Scenario: Collision resolution

- GIVEN skill "go-testing" in both the global and profile layers
- WHEN the index is built
- THEN exactly one entry exists, sourced from the profile layer

#### Scenario: Layered aggregation

- GIVEN distinct skills in each of the four layers (global, remote, project, profile)
- WHEN the index is built
- THEN all skills are present

#### Scenario: Remote vs local collision

- GIVEN skill "go-testing" in both the remote layer and the project layer
- WHEN the index is built
- THEN exactly one entry exists, sourced from the project layer

#### Scenario: Remote never shadows profile

- GIVEN skill "go-testing" in both the remote layer and the profile layer
- WHEN the index is built
- THEN exactly one entry exists, sourced from the profile layer

### Requirement: REQ-SKILL-2 — Index and Trigger Matching

Each skill MUST expose a name, description, and triggers. A skill MUST be applicable when a message matches at least one trigger. The index MUST be buildable without reading any skill body.
(Previously: No change to trigger matching logic itself.)

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
(Previously: No change to content loading behavior.)

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

## ADDED Requirements

### Requirement: REQ-RS-13 — Skills Field Accepts URLs

The `skills: []string` field in profile config MUST accept HTTP/HTTPS URLs alongside local skill names. URLs point to registry `index.json` endpoints.

#### Scenario: Mixed local and remote entries

- GIVEN a profile with `skills: ["go-testing", "https://example.com/skills/index.json"]`
- WHEN the index is built
- THEN "go-testing" is resolved locally
- AND skills from the registry URL are fetched remotely

#### Scenario: All URLs

- GIVEN a profile with only registry URLs in `skills`
- WHEN the index is built
- THEN only remote skills are indexed (plus global/project layer skills)

### Requirement: REQ-RS-14 — URL vs Path Classification

Entries in `skills: []string` MUST be classified as remote (http/https URLs) or local (everything else). Classification MUST happen at index build time.

#### Scenario: HTTP URL classified as remote

- GIVEN entry "https://example.com/skills/index.json"
- WHEN classified
- THEN it is marked as remote

#### Scenario: Local name classified as local

- GIVEN entry "go-testing"
- WHEN classified
- THEN it is marked as local

### Requirement: REQ-RS-15 — Remote Layer Position

Remote skills MUST layer between global and project in the scan order: `global → remote → project → profile`. This means remote skills can be shadowed by project and profile layers but shadow global skills.

#### Scenario: Remote shadows global

- GIVEN skill "go-testing" in both global and remote layers
- WHEN the index is built
- THEN the remote version is used

#### Scenario: Project shadows remote

- GIVEN skill "go-testing" in both remote and project layers
- WHEN the index is built
- THEN the project version is used

### Requirement: REQ-RS-16 — Remote Skill Name Prefixing

Remote skill names MUST be prefixed with the registry hostname to avoid collisions across registries. The prefix format is `{hostname}/{skillName}`.

#### Scenario: Unique naming across registries

- GIVEN skill "go-testing" from "registry-a.com" and "registry-b.com"
- WHEN both are indexed
- THEN entries are "registry-a.com/go-testing" and "registry-b.com/go-testing"

#### Scenario: No collision with local

- GIVEN local skill "go-testing" and remote skill "go-testing" from "registry.com"
- WHEN both are indexed
- THEN local remains "go-testing" and remote is "registry.com/go-testing"
