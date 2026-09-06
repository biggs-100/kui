# tui-theme-system Specification

## Purpose

The theme system ports pi's `dark.json` as the `pi-dark` default theme, loads themes from JSON, and bans hard-coded hex literals outside `internal/tui/theme/`.

## Requirements

### Requirement: REQ-TUI-THEME-1 — pi-dark Default Theme

The default theme MUST be `pi-dark`, ported exactly from pi `dark.json`: EVERY `vars` + `colors` entry MUST map to a `theme.go` field (mapping table in design); unmapped entries MUST be listed explicitly, never silently dropped. `Theme` keeps 40+ fields.
(Previously: theme "opencode" matching `assets/opencode.json`)

#### Scenario: Default is pi-dark

- GIVEN no theme override
- WHEN `Load("pi-dark")` is inspected
- THEN all mapped fields equal `dark.json` hexes exactly

#### Scenario: Unknown tokens omitted

- GIVEN a markdown/syntax token with no mapping
- WHEN rendered
- THEN output falls back muted, never invents a color

### Requirement: REQ-TUI-THEME-2 — Tint and Derived Colors

System MUST provide `tint(background, foreground, 0.25)` for logo shadow and `selectedForeground`/`generateSyntax`/`generateSystem` equivalents that derive terminal palette fallback and syntax rules from Theme.

#### Scenario: Tint produces shadow

- GIVEN `background=#1a1a1a` and `fg=#e0e0e0`
- WHEN `tint(bg, fg, 0.25)` is called
- THEN result is blended hex distinct from both inputs

#### Scenario: Syntax rules from theme

- GIVEN a Theme
- WHEN `getSyntaxRules(theme)` is called
- THEN rules map `comment/keyword/function/string/number/type/variable/operator/punctuation` to theme syntax colors

### Requirement: REQ-TUI-THEME-3 — JSON Loader

System MUST load `Theme` from JSON file via `ParseFile`/`ParseBytes` and via discovery `Discover(dirs)` scanning `themes/*.json`. Discovery MUST prefer later dirs overriding earlier.

#### Scenario: Parse opencode.json

- GIVEN valid `opencode.json` bytes
- WHEN `ParseBytes` is called
- THEN it returns Theme without error

#### Scenario: Discovery finds file themes

- GIVEN `t.TempDir()/themes/custom.json`
- WHEN `Discover` runs
- THEN `custom` theme is available via `Load("custom")`

### Requirement: REQ-TUI-THEME-4 — No Hex Literals Outside Theme

System MUST contain zero hard-coded hex literals outside `internal/tui/theme/`. `parity_test.go` MUST fail on any `#[0-9a-fA-F]{6}` outside theme package. Residuals `#2a2a2a/#252525/#e0af68/#569cd6` MUST become tokens (`BGHighlight/InputBar/CodeBlock/Thought` → `backgroundElement/borderSubtle/warning/primary`).

#### Scenario: Guard bans literals

- GIVEN grep for hex in `internal/tui/{views,ui,markdown,app}/*.go`
- WHEN guard runs
- THEN zero matches or test fails

#### Scenario: Styles use tokens

- GIVEN `styles.CodeBlock` or `InputBar`
- WHEN rendered
- THEN colors reference `theme.*` fields not literals

### Requirement: REQ-TUI-THEME-5 — Background Token Distinction

Styles MUST distinguish `backgroundPanel` (sidebar/panel), `backgroundElement` (input/prompt), `backgroundMenu` (selected list item), `background` (app bg). `Panel` style MUST use `backgroundPanel`; prompt `backgroundElement`; `DialogSelect` selection MUST use `backgroundMenu`+`selectedListItemText`.

#### Scenario: Panel uses backgroundPanel

- GIVEN sidebar rendered
- WHEN view is dumped to text
- THEN panel bg matches `backgroundPanel` not generic `BG`

#### Scenario: Selection uses backgroundMenu

- GIVEN DialogSelect with selection
- WHEN dumped
- THEN selected row uses `backgroundMenu` token

### Requirement: REQ-TUI-THEME-6 — Theme Switching Keeps Working

Runtime theme switching MUST keep working; after the port at least `pi-dark` plus one other theme MUST load and apply without restart artifacts.

#### Scenario: Switch applies

- GIVEN running session on pi-dark
- WHEN user switches theme
- THEN editor border + transcript colors update without panic

#### Scenario: Missing theme file

- GIVEN `Load("nonexistent")`
- WHEN called
- THEN it returns an error and current theme stays active
