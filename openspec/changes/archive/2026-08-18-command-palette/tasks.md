# Tasks: Command Palette

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~310 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | auto-chain |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

## Phase 1: Command Registry

- [x] 1.1 RED — Test `TestCommandRegistryCreate` in `internal/tui/commands_test.go`: registry created with N commands, `All()` returns all entries
- [x] 1.2 GREEN — Implement `CommandRegistry` struct in `internal/tui/commands.go`: `Command` type (Name, Description, Category, Shortcut, Args, Handler), `NewCommandRegistry()` registering all 13 commands with categories/metadata, `All()` and `Lookup(name)` methods
- [x] 1.3 RED — Test `TestRegistryLookup` in `internal/tui/commands_test.go`: finds known command by name, returns nil for unknown name
- [x] 1.4 RED — Test `TestRegistryHelpText` in `internal/tui/commands_test.go`: `HelpText()` returns categorized output with descriptions and shortcuts

## Phase 2: Command Palette View

- [x] 2.1 RED — Test `TestPaletteCreate` in `internal/tui/views/command_palette_test.go`: creates palette with commands, initial state shows all items
- [x] 2.2 GREEN — Implement `CommandPaletteModel` in `internal/tui/views/command_palette.go`: wraps `bubbles/list`, `commandItem` implementing `list.Item` with FilterValue from name+description, `commandItemDelegate` rendering name/description/shortcut columns, `NewCommandPaletteModel(cmds, width, height)`, `Update()`, `View()`, `Selected()`
- [x] 2.3 RED — Test `TestPaletteFilter` in `internal/tui/views/command_palette_test.go`: typing "session" narrows list to session-category commands, typing "reload" shows only reload
- [x] 2.4 RED — Test `TestPaletteEscape` in `internal/tui/views/command_palette_test.go`: Escape returns empty selection

## Phase 3: App Integration

- [x] 3.1 RED — Test `TestAppPaletteToggle` in `internal/tui/app_test.go`: Ctrl+P sets `paletteMode=true`, Escape clears it
- [x] 3.2 GREEN — Wire palette into `internal/tui/app.go`: add `paletteMode bool` + `commandPalette *views.CommandPaletteModel` fields, handle `Ctrl+P` in `handleKey` before `listMode` guard, create palette with registry commands, delegate palette keys when active, render palette in `View()` when active
- [x] 3.3 GREEN — Replace `handleCommand` switch dispatch in `internal/tui/app.go` with `registry.Lookup(name)` + call `Handler(parts)`
- [x] 3.4 GREEN — Replace `defaultCommands` in `internal/tui/autocomplete.go` with registry-derived list via `registry.All()`
- [x] 3.5 RED — Test `TestAppHelpCategorized` in `internal/tui/app_test.go`: sending `/help` produces categorized output with category headers and command descriptions
- [x] 3.6 GREEN — Replace `/help` string in `handleCommand` with `registry.HelpText()` output
- [x] 3.7 GREEN — Update `internal/tui/app_test.go` helper functions to use registry for command strings instead of hardcoded literals

## Phase 4: Verification

- [x] 4.1 Run `go test -race ./...` — full suite passes
- [x] 4.2 Run `go vet ./...` and `golangci-lint run ./...` — no warnings
