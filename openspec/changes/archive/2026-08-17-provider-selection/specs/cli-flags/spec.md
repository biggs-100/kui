# Delta for cli-flags

## MODIFIED Requirements

### Requirement: REQ-CLI-10 — Options Struct

The system MUST define an `Options` struct holding all parsed flag values: `Provider string`, `Model string`, `Tools string`, `ExcludeTools string`, `NoTools bool`, `NoExtensions bool`, `NoSkills bool`, `NoSession bool`, `Verbose bool`, `Mode string`, `Approve bool`, `Print bool`.

(Previously: No `Provider` field in Options)

#### Scenario: All fields default to zero values

- GIVEN no flags provided
- WHEN `parseFlags` is called with empty args
- THEN all Options fields are their zero values (empty string or false)

#### Scenario: Provider flag set

- GIVEN args `["--provider", "openai"]`
- WHEN `parseFlags` is called
- THEN `Options.Provider == "openai"`

#### Scenario: Provider short flag

- GIVEN args `["-p", "opencode"]`
- WHEN `parseFlags` is called
- THEN `Options.Provider == "opencode"`
