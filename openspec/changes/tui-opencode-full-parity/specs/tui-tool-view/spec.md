# Delta for tui-tool-view

## MODIFIED Requirements

### Requirement: REQ-TUI-TOOL-1 — Live Tool Events

The tool view MUST render each tool call and result as events arrive. It MUST support `collapseToolOutput` and `genericToolOutput` toggle via kv signals `showDetails/showGenericToolOutput`. Collapsed output MUST truncate with expand hint. Each entry MUST show per-tool metadata, not just `Name → Result` in rounded panel.
(Previously: per-entry `Panel` rounded `#252525/#333`, `○ pending` only)

#### Scenario: Pending then result

- GIVEN `read_file` pending then result `ok`
- WHEN tool View dumps
- THEN first shows `○ pending`, then `Result: ok`

#### Scenario: Collapse truncates

- GIVEN long output 500 lines with `collapseToolOutput=true`
- WHEN rendered
- THEN dump shows truncated preview + `… N lines` hint

#### Scenario: Toggle details

- GIVEN `showDetails=false` then `true`
- WHEN re-rendered
- THEN detail rows appear only in second dump

## ADDED Requirements

### Requirement: REQ-TUI-TOOL-3 — Diff Rendering and File Tree

System MUST render diff via file-tree utils with `CHANGED FILES` + `▶` cursor + `+N/-N`, line numbers with `diffLineNumber*Bg`, `EmptyBorder/SplitBorder` chars, hunk header `diffHunkHeader`, highlight `diffHighlight`/`diff*Bg`, `diffWrapMode` word/none from kv store.

#### Scenario: Diff tree shows counts

- GIVEN diff with 2 files (+10/-2)
- WHEN dumped
- THEN lines contain `▶` and `+10`/`-2`

#### Scenario: Line numbers styled

- GIVEN hunk with line numbers
- WHEN dumped as text
- THEN number column is present and wraps via word mode when enabled

#### Scenario: Wrap mode none truncates

- GIVEN `diff_wrap_mode=none` and long line 200 cols at width 80
- WHEN dumped
- THEN line is truncated not wrapped

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
