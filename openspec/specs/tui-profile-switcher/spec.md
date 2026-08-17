# tui-profile-switcher Specification

## Purpose

The header renders one tab per discoverable profile and lets the user cycle the session-active profile with TAB / shift+TAB. Switches are queued to the steering queue and apply between turns.

## Requirements

### Requirement: REQ-TUI-PROF-1 — Profile Tabs

The header MUST render one tab per discoverable profile (from the profile loader) and MUST visually mark the active profile.

#### Scenario: Tabs render

- GIVEN two discoverable profiles "coder" and "writer"
- WHEN the header renders
- THEN two tabs appear
- AND the active one is visually marked

### Requirement: REQ-TUI-PROF-2 — TAB Cycle with Wrap

TAB MUST advance to the next profile and shift+TAB MUST move to the previous profile. At either end the cycle MUST wrap around. Each TAB press MUST advance exactly one step, so rapid presses cycle deterministically.

#### Scenario: Forward wrap

- GIVEN profile "writer" active in a two-profile set
- WHEN the user presses TAB
- THEN "coder" becomes active (wrapped from the last tab)

#### Scenario: Backward wrap

- GIVEN profile "coder" active
- WHEN the user presses shift+TAB
- THEN "writer" becomes active

#### Scenario: Rapid presses

- GIVEN a two-profile set starting at "coder"
- WHEN the user presses TAB three times quickly
- THEN the active profile is "writer"
- AND each press advanced exactly one step

### Requirement: REQ-TUI-PROF-3 — Session-Active Switch Semantics

A TAB switch MUST be enqueued via the steering queue (REQ-PROFILE-3) and MUST apply between turns (REQ-LOOP-5) — never mid-tool-call and never mid-response — with history preserved and a profile-context marker inserted (REQ-LOOP-6). The active profile MUST be session-scoped; a switch MUST NOT persist globally.

#### Scenario: Switch during active turn

- GIVEN a turn in progress and a TAB press
- WHEN the turn completes
- THEN the new profile applies before the next provider request
- AND the conversation history is unchanged

#### Scenario: Switch mid-tool-call

- GIVEN a tool executing and a TAB press
- WHEN the tool call is running
- THEN the active profile does not change until the tool call completes

#### Scenario: Session scoping

- GIVEN a switch to "writer" in the TUI session
- WHEN the app exits and a new CLI invocation starts
- THEN the global active profile is unchanged

### Requirement: REQ-TUI-PROF-4 — No Profiles Fallback

When no profiles are discoverable, the header MUST render a hint that no profiles are available, MUST NOT crash, and the session MUST fall back to the default profile.

#### Scenario: Empty profile set

- GIVEN no discoverable profiles
- WHEN the header renders
- THEN a no-profiles hint renders
- AND the app keeps running with the default profile
