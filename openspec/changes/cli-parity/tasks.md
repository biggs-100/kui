# Tasks: CLI Flags for Pi Parity

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 500-600 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Slice A → Slice B → Slice C |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Flag parser + Options struct | PR 1 | `go test ./cmd/kui/... -run TestParseFlags` | N/A — pure function, no runtime | `cmd/kui/flags.go`, `cmd/kui/flags_test.go` |
| 2 | Model override + tool filtering | PR 2 | `go test ./cmd/kui/... -run TestFilterTools` | N/A — unit tests only | `cmd/kui/main.go` filterTools wiring |
| 3 | Feature disable + output + approve | PR 3 | `go test ./cmd/kui/... -run TestRunPrompt` | `kui --mode json "hello"` | `cmd/kui/main.go` output/feature wiring |

## Slice A: Flag Parser + Options (PR 1)

### Phase 1: Foundation — Options Struct

- [x] A1.1 RED: Write failing test `TestOptionsZeroValues` in `cmd/kui/flags_test.go` — verify all Options fields are zero values when called with empty args
- [x] A1.2 GREEN: Create `cmd/kui/flags.go` with `Options` struct (Model, Tools, ExcludeTools, NoTools, NoExtensions, NoSkills, NoSession, Verbose, Mode, Approve, Print) and empty `parseFlags(args []string) (Options, []string, error)`
- [x] A1.3 REFACTOR: Extract Options field docs into struct comments

### Phase 2: Core Parser

- [x] A2.1 RED: Write failing tests for long flags with space: `TestParseFlagsLongFlagSpace` — `["--model", "gpt-4o"]` → Model="gpt-4o"
- [x] A2.2 RED: Write failing tests for long flags with equals: `TestParseFlagsLongFlagEquals` — `["--model=gpt-4o"]` → Model="gpt-4o"
- [x] A2.3 RED: Write failing tests for boolean flags: `TestParseFlagsBoolFlag` — `["--verbose"]` → Verbose=true
- [x] A2.4 RED: Write failing tests for short flags: `TestParseFlagsShortFlag` — `["-m", "gpt-4o"]` → Model="gpt-4o"
- [x] A2.5 RED: Write failing tests for `--` separator: `TestParseFlagsSeparator` — `["--model", "gpt-4o", "--", "--verbose"]` → Model="gpt-4o", remaining=["--verbose"]
- [x] A2.6 GREEN: Implement `parseFlags` — iterate args, handle `--` separator, parse long/short flags, return (Options, []string, error)
- [x] A2.7 REFACTOR: Extract flag lookup into helper `flagValue(args, i, name) string`

### Phase 3: Error Handling

- [x] A3.1 RED: Write failing tests for unknown flags: `TestParseFlagsUnknownFlag` — `["--unknown-flag"]` → error containing "unknown-flag"
- [x] A3.2 RED: Write failing tests for missing values: `TestParseFlagsMissingValue` — `["--model"]` → error indicating missing value
- [x] A3.3 GREEN: Add unknown-flag and missing-value error returns to parseFlags
- [x] A3.4 REFACTOR: Consolidate error messages into consistent format

## Slice B: Model + Tool Filtering (PR 2)

### Phase 4: Model Override

- [x] B4.1 RED: Write failing test `TestResolveModelOverride` — verify `--model gpt-4o` takes precedence over saved model in resolveModel chain
- [x] B4.2 GREEN: Modify `resolveModel` signature to accept `override string` as highest-priority param; if non-empty, return it immediately
- [x] B4.3 GREEN: Update `runPrompt` to pass `opts.Model` to `resolveModel`
- [x] B4.4 REFACTOR: Add comment documenting override position in resolution chain

### Phase 5: Tool Filtering

- [x] B5.1 RED: Write failing test `TestFilterToolsInclude` — `--tools read_file` → only read_file in registry
- [x] B5.2 RED: Write failing test `TestFilterToolsExclude` — `--exclude-tools bash` → bash removed
- [x] B5.3 RED: Write failing test `TestFilterToolsNoTools` — `--no-tools` → empty registry
- [x] B5.4 RED: Write failing test `TestFilterToolsExcludeWins` — `--tools read_file,bash --exclude-tools bash` → only read_file
- [x] B5.5 GREEN: Implement `filterTools(full *core.Registry, include, exclude string, noTools bool) *core.Registry` in `cmd/kui/flags.go`
- [x] B5.6 GREEN: Wire `filterTools` into `runPrompt` after tool registration, before agent creation
- [x] B5.7 REFACTOR: Extract tool name splitting into helper

## Slice C: Feature Disable + Output (PR 3)

### Phase 6: Feature Disable

- [ ] C6.1 RED: Write failing test `TestNoExtensions` — verify `extensions.LoadAll()` not called when `--no-extensions` set (use spy/counter)
- [ ] C6.2 RED: Write failing test `TestNoSkills` — verify `skills.NewIndex()` not called when `--no-skills` set
- [ ] C6.3 GREEN: Add conditional guards in `runPrompt` for `opts.NoExtensions` and `opts.NoSkills`
- [ ] C6.4 GREEN: Accept `Options` parameter in `runTUI` for feature-disable flags
- [ ] C6.5 REFACTOR: Document no-op `--no-session` in usage string

### Phase 7: Output & Approve

- [ ] C7.1 RED: Write failing test `TestModeJson` — `--mode json` wraps answer in `{"answer":"..."}`
- [ ] C7.2 RED: Write failing test `TestModeJsonRejectTUI` — `--mode json` with `tui` subcommand → error
- [ ] C7.3 RED: Write failing test `TestVerboseStderr` — `--verbose` writes debug info to stderr
- [ ] C7.4 RED: Write failing test `TestApproveWarning` — `--approve` writes warning to stderr
- [ ] C7.5 GREEN: Implement JSON output wrapper at `runPrompt` return boundary
- [ ] C7.6 GREEN: Add `Manager.SetRuleset(*permissions.Ruleset)` method for `--approve` bypass
- [ ] C7.7 GREEN: Wire `--verbose` stderr logging in `runPrompt`
- [ ] C7.8 REFACTOR: Update usage string with all 11 flags

## Phase 8: Integration Testing

- [ ] 8.1 Write integration test `TestRunPromptWithModel` — mock provider, verify model override flows through
- [ ] 8.2 Write integration test `TestRunPromptWithTools` — verify tool filtering affects agent behavior
- [ ] 8.3 Write integration test `TestRunPromptJsonOutput` — verify JSON envelope on stdout
- [ ] 8.4 Run `go test ./cmd/kui/...` — all tests pass
- [ ] 8.5 Run `go vet ./cmd/kui/...` — no issues
