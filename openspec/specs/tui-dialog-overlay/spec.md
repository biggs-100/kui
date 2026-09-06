# tui-dialog-overlay Specification

## Purpose

Overlays (command palette, model/session/provider lists, login, `/status`) render as pi-style centered modal dialogs over the transcript, with a transient status line replacing toasts.

## Requirements

### Requirement: REQ-TUI-DLG-1 — Dialog Overlay Primitive

Selectors MUST render as CENTERED overlays estilo pi (dim backdrop, centered via `lipgloss.Place`, modal keymap layer, `Esc` close with selection guard). Dialogs MUST fit narrow terminals without overflow.
(Previously: `ui/dialog` overlay sizes 60/88/116 with top padding height/4)

#### Scenario: Centered overlay

- GIVEN 120x30 terminal with palette open
- WHEN dumped
- THEN content is centered over the dimmed transcript

#### Scenario: Narrow overlay fits

- GIVEN 60x20 terminal with model list open
- WHEN dumped
- THEN dialog fits without overflow or panic

### Requirement: REQ-TUI-DLG-2 — DialogSelect Grouped Select

System MUST provide generic `DialogSelect` with: `fuzzysort` weighted `title*2+category`, grouping by `category`, `backgroundMenu` selection + `selectedForeground` vs `textMuted` detail, scrollAcceleration, sticky bottom, details `truncateMiddle(76)`, highlight splitting, emptyView.

#### Scenario: Weighted fuzzy sort

- GIVEN items with titles and categories
- WHEN filter "mod" typed
- THEN `model` titles rank above category-only matches

#### Scenario: Grouping renders category header

- GIVEN items with distinct categories
- WHEN rendered
- THEN dump shows `Category` header before its items

#### Scenario: Selected row uses backgroundMenu

- GIVEN selection index 2
- WHEN dumped as text (no color)
- THEN `> ` marker + truncated detail at 76 cols visible

### Requirement: REQ-TUI-DLG-3 — Palette, Model, Session, Provider, Login, Status Dialogs

Command palette, model/session/provider lists, login and `/status` (MCP/LSP dots + error detail) MUST all use the centered overlay. Status MUST show MCP/LSP states from live data only; absent data MUST be omitted muted, never fabricated.
(Previously: palette/model/status via DialogSelect with nano-disable, Free-cost, suggested-grouping specifics)

#### Scenario: Status shows live dots

- GIVEN one MCP server failed with error string
- WHEN `/status` dumps
- THEN failed dot + error detail appear; absent servers are omitted

#### Scenario: Empty filter result

- GIVEN filter `zzz` matching no models
- WHEN dumped
- THEN emptyView renders without crash

### Requirement: REQ-TUI-DLG-4 — Filter-Then-Close Interaction

System MUST clear filter on first Esc when filter non-empty; second Esc closes dialog. Filter `InputRenderable` MUST be focused when dialog open. `preserveSelection` MUST keep selection after filter change with double-rAF re-scroll semantics approximated via sticky selection.

#### Scenario: Esc clears then closes

- GIVEN dialog with filter "foo"
- WHEN Esc pressed once
- THEN filter clears and dialog stays open
- WHEN Esc pressed again
- THEN dialog closes

#### Scenario: Verification via text dump

- GIVEN palette/model/status dialogs
- WHEN `View()` dumped to `testdata/dialog_*.txt`
- THEN goldens match without PNG and pass in `go test`

### Requirement: REQ-TUI-DLG-5 — Toasts Removed, Transient Status Line

Toasts MUST NOT render. Transient messages MUST appear as a single status line in the status container above the editor and auto-clear; the layout MUST NOT jump (`IdleStatus` reserves the lines).

#### Scenario: Error goes to status line

- GIVEN a tool error event
- WHEN dumped
- THEN no floating toast appears; the transient line shows the message

#### Scenario: Layout does not jump

- GIVEN idle then transient message
- WHEN dumped in both states
- THEN editor position is identical (reserved lines)
