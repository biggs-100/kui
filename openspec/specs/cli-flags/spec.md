# cli-flags Specification

## Purpose

Hand-rolled flag parser for kui CLI. Provides `--flag value`, `--flag=value`, `--bool-flag`, and `-short` syntax without stdlib `flag` or external dependencies. Returns parsed options and remaining positional args.

## Requirements

### Requirement: REQ-CLI-6 — Hand-Rolled Flag Parser

The system MUST implement a hand-rolled flag parser in `cmd/kui/flags.go` with no external dependencies and no reliance on the `flag` stdlib package.

#### Scenario: No external dependencies

- GIVEN the `cmd/kui/flags.go` source file
- WHEN imported packages are inspected
- THEN only stdlib packages and internal kui packages are imported
- AND no `flag` stdlib package is used

#### Scenario: Parser compiles and passes tests

- GIVEN the flag parser implementation
- WHEN `go test ./cmd/kui/...` runs
- THEN all tests pass

### Requirement: REQ-CLI-7 — parseFlags Signature

The parser MUST expose `parseFlags(args []string) (Options, []string)` that returns a parsed `Options` struct and remaining positional arguments (the prompt).

#### Scenario: Flags separated from prompt

- GIVEN args `["--model", "gpt-4o", "hello world"]`
- WHEN `parseFlags` is called
- THEN the returned Options has `Model == "gpt-4o"`
- AND the remaining args are `["hello world"]`

#### Scenario: No flags provided

- GIVEN args `["hello world"]`
- WHEN `parseFlags` is called
- THEN the returned Options has zero values
- AND the remaining args are `["hello world"]`

### Requirement: REQ-CLI-8 — Flag Syntax Support

The parser MUST support `--long-flag value`, `--long-flag=value`, `--bool-flag` (no value), and `-s value` short forms.

#### Scenario: Long flag with space

- GIVEN args `["--model", "gpt-4o"]`
- WHEN `parseFlags` is called
- THEN `Options.Model == "gpt-4o"`

#### Scenario: Long flag with equals

- GIVEN args `["--model=gpt-4o"]`
- WHEN `parseFlags` is called
- THEN `Options.Model == "gpt-4o"`

#### Scenario: Boolean flag without value

- GIVEN args `["--verbose"]`
- WHEN `parseFlags` is called
- THEN `Options.Verbose == true`

#### Scenario: Short flag

- GIVEN args `["-m", "gpt-4o"]`
- WHEN `parseFlags` is called
- THEN `Options.Model == "gpt-4o"`

#### Scenario: Flags stop at `--`

- GIVEN args `["--model", "gpt-4o", "--", "--verbose"]`
- WHEN `parseFlags` is called
- THEN `Options.Model == "gpt-4o"`
- AND remaining args are `["--verbose"]`

### Requirement: REQ-CLI-9 — Unknown Flag Error

The parser MUST return an actionable error message when an unknown flag is encountered, naming the unrecognized flag.

#### Scenario: Unknown long flag

- GIVEN args `["--unknown-flag"]`
- WHEN `parseFlags` is called
- THEN an error is returned containing "unknown-flag"

#### Scenario: Unknown short flag

- GIVEN args `["-z"]`
- WHEN `parseFlags` is called
- THEN an error is returned containing "-z"

### Requirement: REQ-CLI-10 — Options Struct

The system MUST define an `Options` struct holding all parsed flag values: `Provider string`, `Model string`, `Tools string`, `ExcludeTools string`, `NoTools bool`, `NoExtensions bool`, `NoSkills bool`, `NoSession bool`, `Verbose bool`, `Mode string`, `Approve bool`, `Print bool`.

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

#### Scenario: Partial flags set

- GIVEN args `["--verbose", "--model", "gpt-4o"]`
- WHEN `parseFlags` is called
- THEN `Options.Verbose == true` and `Options.Model == "gpt-4o"`
- AND other fields remain at zero values
