# Design: Command Palette

## Technical Approach

Extract command metadata into a centralized `CommandRegistry` that drives both the palette view and `/help`. The palette follows the existing `SessionListModel` pattern — a Bubble Tea list component with custom delegate, toggled via an App-level mode flag. No new dependencies; `sahilm/fuzzy` (already indirect via bubbles) powers search.

## Architecture Decisions

### Decision: CommandRegistry as the single source of truth

**Choice**: Standalone `CommandRegistry` struct in `internal/tui/commands.go` holding all command metadata + handler function pointers.

**Alternatives considered**: Keep `handleCommand` switch and add a parallel metadata map. Refactor into per-command handler files.

**Rationale**: A registry eliminates duplication (autocomplete list, help text, palette, and dispatch all derive from one structure). Handler functions as `func(parts []string) tea.Cmd` keep execution logic colocated with metadata. The switch in `handleCommand` is replaced by a lookup + call.

### Decision: App-level mode flag for palette (not embedding in session list)

**Choice**: Add `paletteMode bool` + `commandPalette *views.CommandPaletteModel` to `App`, mirroring the existing `listMode`/`sessionList` pair. Only one modal active at a time.

**Alternatives considered**: Stack multiple modals. Embed palette inside input model.

**Rationale**: Matches the proven session-list pattern. Single-mode enum would be cleaner but the existing codebase already uses paired bool+pointer — changing that is out of scope. Guard with `if a.listMode` before checking `a.paletteMode`.

### Decision: Inline fuzzy matching (no external fuzzy library)

**Choice**: Use `sahilm/fuzzy` directly (already in go.sum as bubbles dependency) for matching command name + description.

**Alternatives considered**: Simple substring match. Port a Go fuzzy library.

**Rationale**: `sahilm/fuzzy` is already vendored through bubbles. Substring matching is too rigid for discovery ("switch" won't match "Switch profile"). Fuzzy scoring gives ranked results with minimal code.

### Decision: Which-key deferred

**Choice**: Skip which-key overlay in this change. The palette itself solves discoverability.

**Alternatives considered**: Build a transient key-chord overlay.

**Rationale**: Which-key timing across terminal emulators is fragile. Ctrl+P + palette provides the same discovery path. Add which-key later as a separate change.

## Data Flow

```
User presses Ctrl+P
    │
    ▼
App.handleKey ──→ paletteMode = true
    │               commandPalette created with registry commands
    ▼
View() ──→ commandPalette.View() (replaces normal layout)
    │
    ▼
User types in palette search
    │
    ▼
commandPalette.Update() ──→ fuzzy filter on name + description
    │                        grouped by category
    ▼
User presses Enter
    │
    ▼
commandPalette.Selected() ──→ App executes handler from registry
    │                          paletteMode = false
    ▼
Normal TUI resumes
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/commands.go` | Create | `Command` struct, `CommandRegistry` with all 13 commands, `Lookup(name)`, `All()`, `HelpText()` methods |
| `internal/tui/commands_test.go` | Create | Unit tests for registry: lookup, help text generation, all commands registered |
| `internal/tui/views/command_palette.go` | Create | `CommandPaletteModel` — wraps `bubbles/list`, fuzzy filter, category delegate, `Selected()` returns command name |
| `internal/tui/views/command_palette_test.go` | Create | Unit tests: fuzzy filtering, selection, escape handling, category grouping |
| `internal/tui/app.go` | Modify | Add `paletteMode`/`commandPalette` fields; wire `Ctrl+P` in `handleKey`; add palette key delegation block; replace `/help` string with `registry.HelpText()`; replace `handleCommand` switch with registry lookup |
| `internal/tui/app_test.go` | Modify | Tests for palette open/close, command execution via palette, Ctrl+P binding |
| `internal/tui/autocomplete.go` | Modify | Replace `defaultCommands` slice with registry-derived list (keeps autocomplete working, removes duplication) |

## Interfaces / Contracts

```go
// internal/tui/commands.go
type Command struct {
    Name        string
    Description string
    Category    string
    Shortcut    string // empty if none
    Args        string // usage hint, e.g. "<session-id>"
    Handler     func(parts []string) tea.Cmd
}

type CommandRegistry struct {
    commands []Command
}

func NewCommandRegistry() *CommandRegistry
func (r *CommandRegistry) Lookup(name string) *Command
func (r *CommandRegistry) All() []Command
func (r *CommandRegistry) HelpText() string
```

```go
// internal/tui/views/command_palette.go
type CommandPaletteModel struct { /* wraps bubbles/list */ }
func NewCommandPaletteModel(cmds []Command, width, height int) CommandPaletteModel
func (m CommandPaletteModel) Update(msg tea.Msg) (CommandPaletteModel, tea.Cmd)
func (m CommandPaletteModel) View() string
func (m CommandPaletteModel) Selected() string // returns command name, empty if dismissed
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Registry lookup, help text, all commands registered | `go test ./internal/tui/ -run TestRegistry` |
| Unit | Palette fuzzy filtering, selection, escape | `go test ./internal/tui/views/ -run TestPalette` |
| Unit | Palette category grouping | Verify commands grouped by Category field |
| Integration | Ctrl+P opens palette, Enter executes, Escape closes | `go test ./internal/tui/ -run TestAppPalette` |
| Integration | `/help` shows categorized output | `go test ./internal/tui/ -run TestHelp` |
| E2E | Full `go test -race ./...` | CI gate |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. This is purely additive UI code. The existing `handleCommand` switch is replaced by registry dispatch but produces identical behavior. Autocomplete gets its command list from the registry, so `defaultCommands` in `autocomplete.go` is removed — no external callers depend on it.

## Open Questions

- [ ] Should the palette show keyboard shortcuts for commands that have them (Tab, d) even though those are app-level keys, not slash commands? Proposal says yes — include them as informational rows.
- [ ] Palette renders as full-screen overlay (like session list) or as a centered modal floating over the chat? Proposal suggests session-list pattern — full replacement of View(). Recommend sticking with that for consistency.
