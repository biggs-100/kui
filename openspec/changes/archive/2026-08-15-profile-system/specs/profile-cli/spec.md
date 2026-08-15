# profile-cli Specification

## Purpose

The profile CLI exposes profile management on the `kui` binary: listing profiles, switching the active profile, and per-profile model selection. Model choices persist across sessions; all subcommands fail loudly with actionable errors.

## Requirements

### Requirement: REQ-PCLI-1 — List Profiles

`kui profile list` MUST enumerate the resolved profiles and MUST mark the active one. With no profiles, it MUST exit zero and print an empty list.

#### Scenario: List with active marker

- GIVEN two profiles with "coder" active
- WHEN `kui profile list` runs
- THEN both profiles print
- AND "coder" is marked as active
- AND the exit status is zero

#### Scenario: No profiles

- GIVEN no profile.yaml anywhere
- WHEN `kui profile list` runs
- THEN an empty list prints
- AND the exit status is zero

### Requirement: REQ-PCLI-2 — Switch Profile

`kui profile switch <name>` MUST activate the named profile for the session. An unknown profile name MUST produce a non-zero exit and an actionable stderr message naming the profile.

#### Scenario: Switch to known profile

- GIVEN profile "coder" exists
- WHEN `kui profile switch coder` runs
- THEN "coder" becomes the active profile
- AND the exit status is zero

#### Scenario: Switch to unknown profile

- GIVEN `kui profile switch nope` and no such profile
- WHEN the command runs
- THEN stderr names "nope" as unknown
- AND the exit status is non-zero

### Requirement: REQ-PCLI-3 — Per-Profile Model

`kui profile model <name> <model>` MUST set and persist the model for the named profile. Missing arguments MUST produce usage text and a non-zero exit.

#### Scenario: Model set persists

- GIVEN `kui profile model coder gpt-4o`
- WHEN the command runs
- THEN the model is persisted for profile "coder"
- AND the exit status is zero

#### Scenario: Missing arguments

- GIVEN `kui profile model coder` with no model value
- WHEN the command runs
- THEN usage text prints
- AND the exit status is non-zero
