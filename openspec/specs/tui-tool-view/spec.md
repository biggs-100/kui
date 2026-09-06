# tui-tool-view Specification

## Purpose

The tool view renders live tool calls and results during multi-step turns, driven by the agent loop's observer port.

## Requirements

### Requirement: REQ-TUI-TOOL-1 — Live Tool Events

Each tool call MUST render as a `Box` with state bg: pending `#282832` / success `#283228` / error `#3c2828`. Each block MUST show the call title + up to 10-line preview + expand hint (`… N lines`). Toggling expands to full output. Events MUST render as they arrive.
(Previously: rounded Panel `#252525/#333`, kv `showDetails`/`showGenericToolOutput`, `○ pending`)

#### Scenario: Pending then success

- GIVEN `read_file` pending then result ok
- WHEN dumped at each stage
- THEN bg is `#282832` then `#283228` with title + preview

#### Scenario: Long output collapses

- GIVEN 500-line output collapsed
- WHEN rendered
- THEN dump shows 10 lines + `… 490 lines` hint

### Requirement: REQ-TUI-TOOL-2 — Graceful Degradation

When the observer is nil or unavailable, the tool view MUST stay empty/disabled, MUST NOT crash, and MUST NOT alter the loop's behavior.

#### Scenario: Nil observer

- GIVEN a loop running with a nil observer
- WHEN the app renders
- THEN the tool view shows no events
- AND the app keeps running normally

#### Scenario: Observer unavailable mid-turn

- GIVEN an observer that stops delivering events mid-turn
- WHEN the turn completes
- THEN the tool view degrades without crashing
- AND the loop's termination and output are unaffected

### Requirement: REQ-TUI-TOOL-3 — Diff Rendering Inside Block

Diffs MUST render INSIDE the tool `Box` (file header + hunks with line numbers + `+N/-N` counts). A standalone diff panel/overlay MUST NOT exist (`Ctrl+D` panel removed).
(Previously: standalone file-tree view with CHANGED FILES + `▶` cursor + `diffWrapMode` kv)

#### Scenario: Diff inside block

- GIVEN edit diff over 2 files (+10/-2)
- WHEN dumped
- THEN counts appear inside the tool Box, no separate panel

#### Scenario: Narrow diff truncates

- GIVEN width 60 and a 200-col line
- WHEN dumped
- THEN line truncates without panic

### Requirement: REQ-TUI-TOOL-4 — Verification Goldens

All tool and diff states MUST be verified via text-dump goldens at 80/120 cols, no PNG. Goldens include: pending, result, collapsed, diff two-file, diff wrap.

#### Scenario: Golden diff two-file

- GIVEN `testdata/diff_two_file.txt` golden
- WHEN `go test ./internal/tui/views -run TestDiffGolden -update` generates
- THEN diff without update matches locked file

#### Scenario: No fabrication in tool

- GIVEN tool `mcp` result missing
- WHEN rendered
- THEN muted `NotAvailable` not fake `319k` appears
