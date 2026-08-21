# Delta for tui-dialog-overlay

## ADDED Requirements

### Requirement: REQ-TUI-DLG-1 — Dialog Overlay Primitive

System MUST provide `ui/dialog` overlay with: backdrop `RGBA(0,0,0,150)`, sizes `60/88/116` cols, centered via `lipgloss.Place(width,height,Center,Center)`, top padding `height/4`, modal stack pushing `"modal"` keymap layer, Esc/Ctrl+C close with selection guard.

#### Scenario: Overlay centers and dims

- GIVEN terminal 120x30 and dialog size 88
- WHEN `Dialog.View()` renders
- THEN content is centered and overlay char dump shows backdrop

#### Scenario: Modal stack on open

- GIVEN closed app
- WHEN dialog opens
- THEN keymap layer `modal` is pushed

#### Scenario: Esc closes

- GIVEN dialog open with empty filter
- WHEN Esc pressed
- THEN dialog closes and layer popped

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

### Requirement: REQ-TUI-DLG-3 — Palette, Model, Status Dialogs

System MUST implement dialogs via `DialogSelect`: command palette (groups suggested when no filter, hidden excluded, `COMMAND_PALETTE_COMMAND` excluded, bindings via `formatKeyBindings` with leader token), model dialog (favorites/recent/provider sections, `opencode/*-nano` disabled, `Free` for `cost.input==0`, current `●`), status dialog (MCP `•` colored `connected=success/failed=error/disabled=textMuted/needs_auth=warning` + error detail; LSP similarly; Formatters/Plugins `file://` or `name@version`).

#### Scenario: Palette suggested on top

- GIVEN no filter
- WHEN palette renders
- THEN suggested commands appear before others

#### Scenario: Model dialog disables nano

- GIVEN `opencode/gpt-nano` in models
- WHEN model dialog renders
- THEN row is muted and not selectable

#### Scenario: Status dots colored by state

- GIVEN MCP server `failed` with error string
- WHEN status dump rendered
- THEN detail line shows error under muted form

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
