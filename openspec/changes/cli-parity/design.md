# Design: CLI Flags for Pi Parity

## Technical Approach

Add a hand-rolled flag parser in `cmd/kui/flags.go` that returns an `Options` struct and remaining positional args (the prompt). Wire into `run()` before subcommand dispatch. Each flag maps to a single field on `Options`. Filtering and feature-disable are applied post-`runtime.Build()` (or its equivalent in the CLI path) so the core composition path stays unchanged.

The key insight from reading the code: `runPrompt()` in `cmd/kui/main.go` builds the tool registry and agent *directly* — it does NOT go through `runtime.Build()`. The `runtime.Build()` path exists for the TUI reload flow. This means CLI-path modifications are surgical: parse flags early, inject overrides at specific points in the existing `runPrompt` flow.

## Architecture Decisions

| Decision | Option A | Option B | Tradeoff | Decision |
|----------|----------|----------|----------|----------|
| Flag parser | Hand-rolled in `cmd/kui/flags.go` | stdlib `flag` package | stdlib is familiar but `flag` doesn't support `--` separator coexistence cleanly and adds conceptual weight for 11 flags. Hand-rolled is ~100 lines, testable, zero deps | Hand-rolled |
| Tool filtering location | Post-Build at CLI layer | Inside `runtime.Build()` | Inside Build couples runtime to CLI flags. Post-build is composable and spec REQ-CLI-18 requires it | Post-Build CLI layer |
| `--approve` mechanism | Override `Manager` ruleset | Permissive `Ruleset` passed to `Manager` | `Manager.ruleset` is private. Simpler: set `Manager`'s ruleset via new `SetRuleset` method or construct permissive ruleset in `runPrompt` and pass through Manager | New `Manager.SetRuleset(*permissions.Ruleset)` method |
| Model override flow | Inject into `resolveModel` chain | Bypass chain entirely | Chain exists for good reason (profile → env → default). Override should sit above chain, not replace it. Pass `Options.Model` to `resolveModel` as highest-priority param | Highest-priority param |
| `--mode json` output | Wrap at `runPrompt` return | Wrap inside agent | Agent should be format-agnostic. Wrap at the CLI output boundary | CLI output boundary |

## Data Flow

```
args ──→ parseFlags() ──→ Options + prompt
                              │
    ┌─────────────────────────┤
    │                         │
    ▼                         ▼
runPrompt(opts, prompt)    runTUI(opts)
    │                         │
    ├─ openai.NewClient()     ├─ tui.Run(ctx, wiring)
    ├─ tools.Default()        │    ├─ buildComponents()
    ├─ filterTools(registry)  │    ├─ filterTools(registry)  ← opts flow through
    ├─ resolveModel(opts.Model)│   └─ ...
    ├─ agent.Run()            │
    └─ fmt.Fprintln(answer)  └─
         │
         ▼
    opts.Mode == "json"?
    ├─ yes: printf({"answer":"%s"})
    └─ no:  printf(answer)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/kui/flags.go` | Create | `Options` struct + `parseFlags(args) (Options, []string, error)` |
| `cmd/kui/flags_test.go` | Create | Tests for all flag syntaxes, `--` separator, unknown flags, edge cases |
| `cmd/kui/main.go` | Modify | `run()` calls `parseFlags`, passes `Options` to `runPrompt`/`runTUI`; `runPrompt` accepts `Options`; model override injection; tool filtering call; JSON output wrap; verbose stderr |
| `internal/tui/run.go` | Modify | `Run` accepts `Options` (or extended `Wiring`); applies tool filtering and feature-disable flags post-build |
| `internal/agent/profile_manager.go` | Modify | Add `SetRuleset(*permissions.Ruleset)` for `--approve` bypass |

## Interfaces / Contracts

```go
// cmd/kui/flags.go
type Options struct {
    Model        string
    Tools        string   // comma-separated, parsed to []string at use site
    ExcludeTools string   // comma-separated
    NoTools      bool
    NoExtensions bool
    NoSkills     bool
    NoSession    bool   // no-op placeholder
    Verbose      bool
    Mode         string // "text" (default) or "json"
    Approve      bool
    Print        bool
}

func parseFlags(args []string) (Options, []string, error)
```

```go
// cmd/kui/main.go — modified signatures
func runPrompt(root string, opts Options, args []string) int
func runTUI(root string, opts Options) int
```

```go
// internal/agent/profile_manager.go — new method
func (m *Manager) SetRuleset(rs *permissions.Ruleset)
```

```go
// Tool filter — lives in cmd/kui or a small helper
func filterTools(full *core.Registry, include, exclude string, noTools bool) *core.Registry
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `parseFlags` — all 11 flags, syntax variants, `--` separator, unknown flags, missing values | Table-driven tests in `cmd/kui/flags_test.go` |
| Unit | `filterTools` — include only, exclude only, both, no-tools, empty registry | Table-driven, pure function |
| Unit | `Manager.SetRuleset` — overrides existing ruleset | Test with existing manager tests |
| Integration | `runPrompt` with `--model`, `--tools`, `--mode json`, `--verbose` | Mock provider, verify output on stdout/stderr |
| Integration | `runTUI` with `--no-extensions`, `--no-skills` | Verify `LoadAll` not called (spy or counter) |
| E2E | `kui --model gpt-4o "hello"` uses correct model | Requires live provider (manual or CI with key) |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Kui's CLI is a direct execution path: parse flags → build runtime → run agent loop → output answer.

## Migration / Rollout

No migration required. Flags are purely additive — all default to zero values (false/empty string), preserving existing behavior. The `--no-session` flag is a documented no-op. Rollback: delete `cmd/kui/flags.go`, revert `main.go` and `tui/run.go` changes.

## Open Questions

- [ ] Should `Options.Tools` be `string` (comma-separated) or `[]string` (parsed at parse time)? Spec says comma-separated input — keeping as `string` and splitting at use site matches the hand-rolled parser simplicity.
- [ ] TUI path: should `--verbose` write to stderr while TUI is running? Bubble Tea uses alternate screen — stderr writes would be invisible. Design: `--verbose` in TUI mode writes to a log file instead, or is silently ignored. Recommend: log to stderr (visible in terminal multiplexer contexts like tmux).
