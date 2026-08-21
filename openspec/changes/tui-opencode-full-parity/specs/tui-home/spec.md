# Delta for tui-home

## MODIFIED Requirements

### Requirement: REQ-TUI-HOME-1 — Centered Layout

The home screen MUST render logo + prompt + footer vertically centered via flex spacers with `flexGrow` (Previously: manual `topPad` loop). It MUST use a `height 4` spacer and center column of `maxWidth 75` or `70%` auto. All elements MUST remain horizontally centered on resize. Toast MUST be inside centered column.
(Previously: topPad calc, placeHorizontal only)

#### Scenario: Flex spacer centering

- GIVEN terminal 120x30
- WHEN home renders
- THEN dump shows equal top/bottom spacer lines within ±1

#### Scenario: Resize keeps centering

- GIVEN home visible at 80x24 then resized to 160x40
- WHEN dumped
- THEN logo remains centered in both goldens

### Requirement: REQ-TUI-HOME-2 — ASCII Logo

The home screen MUST display two-sided `█▀▀█` logo via left/right pairs with `tint(background, fg, 0.25)` shadow (Previously: single 6-line `██╗` block, single `LogoAccent`). Logo MUST use theme `syntax*` derived colors, not hard-coded.

#### Scenario: Logo has shadow tint

- GIVEN theme `OpenCode`
- WHEN logo renders
- THEN two-tone output is produced (shadow differs from fg)

#### Scenario: Logo tint is theme-derived

- GIVEN custom theme loaded from JSON
- WHEN logo rerenders
- THEN shadow recomputes via `tint`

### Requirement: REQ-TUI-HOME-3 — Bordered Prompt

The home screen MUST display prompt input with maxWidth `75` (absolute) or `auto=70%` of width (Previously: `width-20` capped 70, rounded border `HomeBorder`). Border MUST be `backgroundElement`+`SplitBorder` with `EmptyBorder` decorative bottom `▀`. Placeholder MUST be random from `placeholders` pool (not single `Ask kui…`). MaxHeight MUST be `max(6, height/3)` or config.

#### Scenario: Prompt maxWidth 75 at wide

- GIVEN terminal 160 cols
- WHEN home prompt renders
- THEN prompt width is 75 not 70

#### Scenario: Prompt auto 70% at narrow

- GIVEN terminal 80 cols
- WHEN home prompt renders
- THEN prompt width is ~56

#### Scenario: Placeholder pool

- GIVEN home prompt empty
- WHEN rendered multiple times
- THEN placeholder varies across pool entries

### Requirement: REQ-TUI-HOME-4 — Minimal Footer

The home screen footer MUST be empty plus `home_bottom` plugin slot (muted `NotAvailable` when absent). It MUST NOT show fabricated `dir • LSP ○/● • MCP ○/●` invention. If backing `sync.data.lsp/mcp` absent, footer MUST omit counts as muted.
(Previously: `directory • LSP • MCP • /status` minimal)

#### Scenario: Home footer empty when no slot

- GIVEN no plugin slot and no sync data
- WHEN home footer renders
- THEN dump shows empty or muted placeholder, not `• LSP`

## ADDED Requirements

### Requirement: REQ-TUI-HOME-5 — Header Suppression and Shell Mode

Home route MUST NOT render header tabs. Prompt MUST support `!` shell mode (offset 0 triggers shell style), extmarks virtual text for `● [File]/[Image]/[Pasted ~N lines]` as muted `NotAvailable` when editor/workspace store absent. No plugin slot content MAY be fabricated.

#### Scenario: Home has no header

- GIVEN route home
- WHEN `app.View()` dumps at 120 cols
- THEN no header tab line appears

#### Scenario: Shell prefix triggers mode

- GIVEN prompt text `!ls`
- WHEN prompt renders
- THEN shell mode indicator appears (text dump contains `!`)

#### Scenario: Verification goldens

- GIVEN home at 80/120/160
- WHEN dumped
- THEN `testdata/home_*.txt` matches OpenCode spacer logic ±1 col
