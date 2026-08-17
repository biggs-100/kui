# Proposal: CLI Flags for Pi Parity

## Intent

kui has zero flag parsing — all CLI is manual `args[0]` dispatch. Pi users expect `--model`, `--tools`, `--verbose`, and other flags for scripting and one-shot overrides. This adds 11 flags (Phase 1) with a hand-rolled parser (no new deps), giving scripting parity with pi's most-used features.

## Scope

### In Scope
- `cmd/kui/flags.go` — minimal flag parser (hand-rolled, no `flag` stdlib, no deps)
- `--model <name>` — override model selection (highest priority in resolution chain)
- `--tools <list>` — restrict tool set (comma-separated names)
- `--exclude-tools <list>` — remove tools from default set
- `--no-tools` — disable all tools (loop runs without tool calls)
- `--no-extensions` — skip extension LoadAll
- `--no-skills` — skip skills index build
- `--verbose` — enable debug output to stderr
- `--mode json` — JSON output for scripting (final answer as `{"answer":"..."}`)
- `--approve` — bypass permission prompts (non-interactive mode)
- `--print` — alias for one-shot stdout behavior (documents current default)
- `--no-session` — no-op placeholder (documents that kui has no session persistence yet)

### Out of Scope
- Session persistence flags (`--continue`, `--resume`) — large subsystem, Phase 2+
- Provider selection (`--provider`) — needs multi-provider architecture
- Thinking levels (`--thinking`) — needs provider extension support
- Dynamic extensions — architecturally incompatible with Go compiled-in model

## Capabilities

### New Capabilities
- `cli-flags`: Flag parser, parsed options struct, flag-to-runtime wiring
- `tool-filtering`: `--tools`, `--exclude-tools`, `--no-tools` filter logic on tool registry

### Modified Capabilities
- `agent-cli`: `run()` and `runPrompt()` accept parsed flags; `--model` inserts into resolution chain at highest priority
- `extension-system`: `--no-extensions` skips `LoadAll()` call
- `profile-skills`: `--no-skills` skips `skills.NewIndex()` call
- `profile-runtime`: `--model` override applied before REQ-CLI-4 resolution chain

## Approach

Hand-roll a `parseFlags(args []string) (*Options, []string)` function in `cmd/kui/flags.go`. Returns parsed options and remaining positional args (the prompt). No dependency on `flag` stdlib — keeps parser predictable for `--` separator coexistence. Wire into `run()` before subcommand dispatch. Each flag maps to a single boolean/string field on the `Options` struct.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/kui/flags.go` | New | Flag parser + Options struct |
| `cmd/kui/main.go` | Modified | `run()` calls `parseFlags`, passes options to `runPrompt`/`runTUI` |
| `internal/adapters/tools/` | Modified | Filter function accepts allow/block lists |
| `internal/adapters/skills/index.go` | Modified | Conditional build skip |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Parser edge cases with `--` separator | Med | Tests: flags before/after `--`, mixed positioning |
| `--approve` bypasses security | High | Warn to stderr; document as non-interactive-only |
| `--mode json` breaks TUI | Low | Reject `--mode json` with `tui` subcommand |

## Rollback Plan

Delete `cmd/kui/flags.go`; revert `main.go` to manual `args[0]` dispatch. No migration — flags are purely additive.

## Dependencies

None external. Internal: `tools.Default()` returns full set, filter applied after.

## Success Criteria

- [ ] `kui --model gpt-4o "hello"` uses gpt-4o (overrides profile)
- [ ] `kui --tools read_file "list files"` only has read_file available
- [ ] `kui --no-extensions "hello"` skips extension loading
- [ ] `kui --mode json "hello"` prints `{"answer":"..."}`
- [ ] `kui -- verbose "hello"` passes `verbose "hello"` as prompt (flags stop at `--`)
- [ ] `go test ./cmd/kui/...` passes
