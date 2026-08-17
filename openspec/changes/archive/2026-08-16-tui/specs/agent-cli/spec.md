# Delta for agent-cli

## ADDED Requirements

### Requirement: REQ-CLI-5 — TUI Subcommand Dispatch

The CLI MUST provide a `kui tui` subcommand that starts the interactive TUI. When startup validation fails (e.g. invalid provider configuration), `kui tui` MUST exit non-zero with an actionable stderr message. The existing one-shot prompt behavior (REQ-CLI-1) MUST remain unchanged.

#### Scenario: Starts the TUI

- GIVEN valid configuration and a reachable provider
- WHEN `kui tui` runs
- THEN the Bubble Tea app starts

#### Scenario: Startup validation failure

- GIVEN invalid provider configuration
- WHEN `kui tui` runs
- THEN an actionable error prints to stderr
- AND the exit status is non-zero

#### Scenario: One-shot prompt unchanged

- GIVEN a prompt argument and a reachable provider
- WHEN the existing one-shot prompt mode runs
- THEN the loop completes and the final answer prints to stdout
- AND the exit status is zero
