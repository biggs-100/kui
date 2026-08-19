# tui-reload Specification

## Purpose

The TUI exposes `/reload` as a slash command, tracks active runs so reload can cancel-and-wait, shows reload status, and refreshes the profile list after a successful reload.

## Requirements

### Requirement: REQ-RELOAD-11 — /reload Slash Command

The TUI input MUST recognize `/reload` as a command, not a prompt. Issuing `/reload` MUST trigger the reload flow instead of sending the text to the agent.

#### Scenario: /reload triggers reload

- GIVEN a running TUI
- WHEN the user types `/reload` and submits
- THEN the reload flow starts
- AND no prompt is sent to the agent

#### Scenario: Other input still prompts

- GIVEN a running TUI
- WHEN the user submits a normal message
- THEN it is sent to the agent as a prompt
- AND no reload occurs

### Requirement: REQ-RELOAD-12 — Reload Status Line

The TUI MUST show a status line during reload: "reloading…" while in progress, then "reload complete: N skills" on success (N = indexed skills), or the reload error on failure.

#### Scenario: Success status

- GIVEN a successful reload with 12 skills indexed
- WHEN the reload completes
- THEN the status line shows "reload complete: 12 skills"

#### Scenario: Failure status

- GIVEN a reload whose build fails
- WHEN the reload completes
- THEN the status line shows the reload error

### Requirement: REQ-RELOAD-13 — Controller Tracks Active Runs

The controller MUST track active runs: running state, cancel, and run-done. Reload MUST cancel-and-wait while a run is tracked as running.

#### Scenario: Run lifecycle tracked

- GIVEN a prompt submitted
- WHEN the run starts
- THEN the controller marks the run as running
- AND marks it done when the run finishes

#### Scenario: Reload waits for a running run

- GIVEN a tracked running run
- WHEN /reload is issued
- THEN the controller cancels the run and waits
- AND proceeds only when the run is marked done

### Requirement: REQ-RELOAD-14 — Reloader Port

The controller MUST expose a `Reloader` port via `SetReloader`, so the TUI can inject the runtime reload function; the controller MUST NOT import the runtime package directly.

#### Scenario: Reloader injected

- GIVEN a controller with SetReloader(r)
- WHEN /reload is issued
- THEN the injected reloader is invoked

#### Scenario: No reloader set

- GIVEN a controller without a reloader
- WHEN /reload is issued
- THEN the TUI shows reload unavailable
- AND the app keeps running

### Requirement: REQ-RELOAD-15 — Profile List Refresh

After a successful reload, the controller's profile list MUST refresh from the reloaded loader, preserving the active profile when it still exists.

#### Scenario: New profile appears

- GIVEN a new profile added on disk
- WHEN a reload succeeds
- THEN the profile list includes the new profile
- AND the active profile is unchanged

#### Scenario: Active profile removed

- GIVEN the active profile deleted on disk
- WHEN a reload succeeds
- THEN the active selection falls back to the first available profile
- AND the app does not crash