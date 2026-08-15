# ADR-0002: Environment-based credentials and configuration

- Status: accepted
- Date: 2026-08-14

## Context

kui needs an API key to talk to a chat-completions provider. Keys must never
be committed, must never appear in error messages or logs, and the CLI must
fail fast with an actionable message when configuration is missing. A config
file would introduce secret-storage, precedence, and format questions before
the first release; the product already plans a profile system in a later
change.

## Decision

All provider configuration comes from environment variables, read once at
provider construction time (`openai.NewClient`):

- `OPENAI_API_KEY` — required; absent or empty fails construction with an
  error naming the variable.
- `OPENAI_BASE_URL` — optional; defaults to `https://api.openai.com/v1`.
- `OPENAI_MODEL` — optional; defaults to `gpt-4o-mini`; every chat request
  carries an explicit model field.

The key never appears in any error message, including errors that wrap
provider responses.

## Alternatives considered

- **Config file under `.kui/`**: natural home later, but adds secret
  handling, merge semantics, and discovery rules now; deferred to the
  profile system.
- **CLI flags**: secrets in shell history and process listings; rejected.
- **OS keychain / secret manager**: platform-dependent and not CI-friendly.
- **`.env` loader**: adds a dependency and file-based secrets; the shell
  already provides environment injection.

## Consequences

- Credentials work identically in local shells, CI, and containers.
- Failure is fast and actionable: `OPENAI_API_KEY is not set` names exactly
  what to fix.
- A future profile system can supersede these variables; until then, env is
  the single source.
- Users must export variables per session or use a shell profile — accepted
  for a CLI-first tool.

## Verification notes

- `client_test.go`: `TestNewClientKeyMissing` (error names the variable),
  `TestChatAuthError` (key never leaks even when the server echoes it).
- CLI smoke (`cmd/kui/main_test.go`): `TestCLIMissingKey` asserts the
  actionable error on stderr and exit code 1.
