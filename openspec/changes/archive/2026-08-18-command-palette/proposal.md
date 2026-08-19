# Proposal: Command Palette

## Intent

kui has 12+ slash commands (`/reload`, `/sessions`, `/resume`, `/quit`, `/exit`, `/help`, `/theme`, `/status`, `/clear`, `/rename`, `/undo`, `/redo`, `/diff`) that users must type exactly. There is no way to discover commands by browsing, no fuzzy search, and no keyboard shortcuts for quick access. A command palette gives users a single entry point to find and execute any command, see available shortcuts, and discover features they didn't know existed.

## Current Gap

The existing command infrastructure is functional but opaque:

- Autocomplete (`internal/tui/autocomplete.go`) filters by prefix only — typing `/re` shows `/reload`, `/resume`, `/rename` but typing `/sessions` alone doesn't reveal related commands like `/resume` or `/rename`.
- `/help` dumps a single flat string to the status bar — no descriptions, no categories, no keyboard shortcuts.
- Keyboard shortcuts are hardcoded (`Tab`/`Shift+Tab` for profile switching, `Ctrl+C` to quit, `q` when input empty, `d` to toggle diff) with no discoverability mechanism.
- No fuzzy search — exact prefix matching only.
- Users cannot find commands by intent ("I want to switch profiles" doesn't lead to `Tab`).

| Capability | Today | Target |
|------------|-------|--------|
| Command discovery | `/help` flat string | Categorized, described list |
| Search | Prefix-only autocomplete | Fuzzy search across name + description |
| Keyboard shortcuts | Undocumented, hardcoded | Shown inline, discoverable via which-key |
| Quick access | Type `/` then command name | Ctrl+P opens palette, search, enter |
| Command descriptions | None | One-line description per command |

## Proposed Solution

### 1. Command Registry

A centralized registry that maps commands to metadata (description, category, keyboard shortcut). Replaces the hardcoded switch in `handleCommand`:

```go
type Command struct {
    Name        string
    Description string
    Category    string
    Shortcut    string // e.g. "Ctrl+P", "Tab" — empty if none
    Args        string // e.g. "<session-id>", "" for no args
    Handler     func(parts []string) tea.Cmd
}

type CommandRegistry struct {
    commands []Command
}
```

Categories group commands: `Session` (`/sessions`, `/resume`, `/rename`), `Edit` (`/undo`, `/redo`, `/clear`), `Runtime` (`/reload`, `/theme`, `/status`), `Navigation` (profile switching, diff toggle), `System` (`/quit`, `/exit`, `/help`).

### 2. Command Palette View

A modal overlay (similar to session list view) triggered by `Ctrl+P`:

- Full-width fuzzy search input at top
- Filtered command list below, grouped by category
- Each row shows: command name, description, keyboard shortcut (if any)
- Arrow keys navigate, Enter executes, Escape dismisses
- Typing filters instantly — fuzzy match on name + description

Implementation follows the `SessionListModel` pattern: a `CommandPaletteModel` in `internal/tui/views/` wrapping a Bubble Tea list with custom delegate.

### 3. Which-Key Overlay

When user presses and holds a modifier key (or enters a key chord prefix), a transient overlay shows available continuations:

- `Ctrl+` held: shows `Ctrl+P: command palette`, `Ctrl+C: quit`
- No modal — renders as a temporary status line at bottom, disappears on key release or timeout (500ms)

This is opt-in for the initial scope — only the command palette shortcut (`Ctrl+P`) gets which-key treatment. Additional shortcuts can be added incrementally.

### 4. Enhanced `/help`

Replace the flat string with categorized output:

```
Session:    /sessions  List and manage sessions
            /resume    Resume a saved session
            /rename    Rename the current session
Edit:       /undo      Undo last conversation turn
            /redo      Redo last undone turn
            /clear     Clear chat display
Runtime:    /reload    Hot-reload runtime state
            /theme     Switch UI theme
            /status    Show current profile status
System:     /quit      Save and exit
            /help      Show this help

Shortcuts:  Ctrl+P  Command palette
            Tab     Switch profile
            d       Toggle diff view
```

## Scope

### In Scope

- `CommandRegistry` struct and initialization with all existing commands
- `CommandPaletteModel` view (`internal/tui/views/command_palette.go`)
- `Ctrl+P` keybinding to open/close palette
- Fuzzy search across command name + description
- Category grouping in palette display
- Keyboard shortcut column in palette
- Enhanced `/help` output using registry
- TUI autocomplete remains unchanged (prefix-based inline completion)

### Out of Scope

- User-defined custom commands / keybindings
- Plugin-contributed commands in the palette (future: dynamic registry)
- Macro recording or command chaining
- Which-key overlay beyond `Ctrl+P` hint (future: full chord display)
- Mouse/click support in palette

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/tui/commands.go` | New | `CommandRegistry`, `Command` type, registration of all commands |
| `internal/tui/views/command_palette.go` | New | `CommandPaletteModel` — Bubble Tea list with fuzzy filter |
| `internal/tui/views/command_palette_test.go` | New | Unit tests for palette filtering, selection, rendering |
| `internal/tui/commands_test.go` | New | Unit tests for registry, command lookup |
| `internal/tui/app.go` | Modified | Wire `Ctrl+P` to palette toggle; delegate palette mode keys; enhance `/help` |
| `internal/tui/app_test.go` | Modified | Tests for palette open/close, command execution via palette |
| `internal/tui/autocomplete.go` | Unchanged | Prefix autocomplete stays as-is |

## Success Criteria

- [ ] `Ctrl+P` opens the command palette overlay
- [ ] Palette shows all commands grouped by category with descriptions and shortcuts
- [ ] Fuzzy search filters commands by name and description as user types
- [ ] Arrow keys navigate, Enter executes selected command, Escape closes palette
- [ ] Executing a command from the palette has the same effect as typing it
- [ ] `/help` shows categorized output with descriptions and keyboard shortcuts
- [ ] Palette does not interfere with normal typing or autocomplete
- [ ] `go test -race ./...` clean
- [ ] New code has ≥80% test coverage

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Palette modal conflicts with session list modal | Medium | Medium | Use a single `mode` enum in App (`normal`, `palette`, `sessionList`) — only one modal active at a time |
| Fuzzy search adds latency on large command sets | Low | Low | Commands are <50 items, linear scan is fine for MVP; can add t/fuzzy later |
| `Ctrl+P` conflicts with terminal paste in some terminals | Medium | Low | `Ctrl+P` is standard for command palettes (VS Code, Sublime); terminals that bind it to paste are rare; fallback: palette also opens via `/palette` |
| Which-key overlay timing is tricky across terminal emulators | Medium | Low | Start without which-key; add only if palette shortcut proves insufficient |
| Category grouping requires future commands to pick a category | Low | Low | Registry validates category at registration time; unknown categories panic in dev |

## Rollback Plan

1. Remove `openspec/changes/command-palette/` directory
2. Delete `internal/tui/commands.go`, `internal/tui/views/command_palette.go`, and their tests
3. Revert `internal/tui/app.go` changes (remove `Ctrl+P` binding, palette mode, enhanced `/help`)
4. No data migration needed — no persistence changes
5. Existing commands and autocomplete remain fully functional

## Dependencies

- Existing `SessionListModel` pattern (`internal/tui/views/session_list.go`) — palette follows same modal approach
- `charmbracelet/bubbles/list` already in go.mod — palette list component reuses it
- No new external dependencies required
