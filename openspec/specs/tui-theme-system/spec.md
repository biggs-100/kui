# Delta for tui-theme-system

## ADDED Requirements

### Requirement: REQ-TUI-THEME-1 — Theme 40+ Fields Parity

System MUST define `Theme` with 40+ fields matching `packages/tui/src/theme/assets/opencode.json`: `primary/secondary/accent/error/warning/success/info/text/textMuted/selectedListItemText/background/backgroundPanel/backgroundElement/backgroundMenu/border/borderActive/borderSubtle/diffAdded/diffRemoved/diffContext/diffHunkHeader/diffHighlight/diffAddedBg/diffRemovedBg/diffContextBg/diffLineNumber*/markdown*(text/heading/link/linkText/code/blockQuote/emph/strong/hRule/listItem) /syntax*(comment/keyword/function/variable/string/number/type/operator/punctuation)/thinkingOpacity` plus base `BG/FG/Border` family.

#### Scenario: Theme has all OpenCode fields

- GIVEN `theme.OpenCode()` or JSON-loaded theme
- WHEN fields are inspected
- THEN all 40+ fields are non-empty and JSON round-trips

#### Scenario: OpenCode JSON matches struct

- GIVEN `packages/tui/src/theme/assets/opencode.json`
- WHEN `theme.ParseBytes` loads it
- THEN no field is lost and hex values equal asset exactly

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
