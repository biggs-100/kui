# reload-concurrency Specification

## Purpose

Reload must never race an active agent run. This domain defines the cancel-and-wait contract, suppression of cancel errors, and manager-level synchronization that makes `ApplySwitch` and `Reload` safe under concurrency.

## Requirements

### Requirement: REQ-RELOAD-6 — Reload Excludes Active Runs

`Reload` MUST NOT proceed while an agent run is active. Reload MUST cancel the active run's context and wait for the run to finish before rebuilding.

#### Scenario: Reload during active run

- GIVEN an agent run in progress
- WHEN /reload is issued
- THEN the run's context is cancelled
- AND Reload waits for the run to complete before rebuilding

#### Scenario: Reload with no active run

- GIVEN no run in progress
- WHEN /reload is issued
- THEN Reload proceeds immediately without cancellation

### Requirement: REQ-RELOAD-7 — Context Cancellation

The active run MUST be cancelled via its context — never by killing goroutines or closing channels.

#### Scenario: Run observes cancellation

- GIVEN a run listening on its context
- WHEN Reload cancels it
- THEN the run exits via context cancellation
- AND the run's goroutine terminates cleanly

#### Scenario: Cancel propagates to provider and tools

- GIVEN a run mid tool-call
- WHEN its context is cancelled
- THEN the tool call and subsequent provider requests observe the cancellation

### Requirement: REQ-RELOAD-8 — Canceled Run Is Not an Error

A run canceled by reload MUST NOT be displayed as an error. The TUI MUST suppress cancel-error display and show the reload status message instead.

#### Scenario: Cancel does not render an error

- GIVEN a run cancelled by reload
- WHEN the run completes
- THEN no error message is shown for the canceled run
- AND the reload status message is displayed

#### Scenario: Genuine errors still display

- GIVEN a run that fails for a non-cancel reason
- WHEN the run completes
- THEN the error is displayed normally

### Requirement: REQ-RELOAD-9 — Manager Mutex

The `agent.Manager` MUST guard `ApplySwitch`, `Registry`, `Ruleset`, `Active`, and `Reload` with a mutex held across each full state mutation, so concurrent calls are safe.

#### Scenario: Concurrent ApplySwitch and Reload are race-free

- GIVEN a manager under `go test -race`
- WHEN ApplySwitch and Reload run concurrently
- THEN no data race is detected
- AND state remains consistent

#### Scenario: ApplySwitch blocks during Reload

- GIVEN Reload holds the lock swapping the registry
- WHEN ApplySwitch is called
- THEN ApplySwitch blocks until Reload releases the lock
- AND applies against the new registry

### Requirement: REQ-RELOAD-10 — Manager.Reload Swap

`Manager.Reload(full)` MUST swap the full tool registry and re-apply the active profile under lock, so the profile's tool subset, permissions, and model reflect the new registry.

#### Scenario: Reload re-applies active profile

- GIVEN a new full registry with a tool added
- WHEN Manager.Reload(full) runs
- THEN the registry is swapped
- AND the active profile's subset includes the new tool
- AND the active profile name is preserved

#### Scenario: Failed re-apply keeps old registry

- GIVEN a profile that fails to resolve against the new registry
- WHEN Manager.Reload runs
- THEN the old registry remains active
- AND the error is returned