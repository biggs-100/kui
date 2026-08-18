# Delta for profile-runtime

## MODIFIED Requirements

### Requirement: REQ-PROFILE-1 — Profile Model

A profile MUST declare name, model, system_prompt (a SYSTEM.md path), tools, skills, permissions, and MAY declare a `provider` string. The loader MUST parse profile.yaml through the loader port and MUST return a typed error naming the offending file on malformed or missing content.

(Previously: No `provider` field in profile declaration)

#### Scenario: Valid profile with provider

- GIVEN profile.yaml with name "coder", model "gpt-4o", provider "openai", and a SYSTEM.md path
- WHEN the loader parses it
- THEN a profile with those values is produced

#### Scenario: Valid profile without provider

- GIVEN profile.yaml with no provider field
- WHEN the loader parses it
- THEN a profile with provider set to empty string is produced

#### Scenario: Malformed yaml

- GIVEN profile.yaml with invalid YAML syntax
- WHEN the loader parses it
- THEN a typed parse error naming the file is returned
