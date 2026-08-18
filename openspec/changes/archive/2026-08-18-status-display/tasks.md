# Tasks: Status Display

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~350 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | single PR |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
400-line budget risk: Low

## Phase 1: Theme Styles

- [x] 1.1 RED — Test: `TestFooterStylesExist` — verify StatusLine/StatusOK/StatusError/StatusWarn styles render non-empty. File: `internal/tui/theme/theme_test.go`
- [x] 1.2 GREEN — Add `StatusLine`, `StatusOK`, `StatusError`, `StatusWarn` fields to `theme.Styles` struct and `NewStyles()` constructor. File: `internal/tui/theme/styles.go`

## Phase 2: FooterModel

- [x] 2.1 RED — Test: `TestNewFooterModel` — creates model, Render returns placeholder. File: `internal/tui/views/footer_test.go`
- [x] 2.2 GREEN — Create `FooterModel` struct with fields (`dir`, `model`, `tokens`, `contextMax`, `cost`, `mcpConnected`, `mcpFailed`, `styles`) and `NewFooterModel(styles)`. File: `internal/tui/views/footer.go`
- [x] 2.3 RED — Test: `TestFooterRenderFull` — all fields set, Render contains each value. File: `internal/tui/views/footer_test.go`
- [x] 2.4 RED — Test: `TestFooterRenderEmpty` — zero state, Render shows placeholder dashes. File: `internal/tui/views/footer_test.go`
- [x] 2.5 GREEN — Implement setter methods: `SetDirectory`, `SetModel`, `SetTokens(tokens, contextMax int)`, `SetCost(cost float64)`, `SetMCPStatus(connected, failed int)`. File: `internal/tui/views/footer.go`
- [x] 2.6 GREEN — Implement `Render() string` — compose `"dir | model | N tokens (P%) | $C | MCP: X/Y"` with lipgloss styles. File: `internal/tui/views/footer.go`

## Phase 3: Controller Token Tracking

- [x] 3.1 RED — Test: `TestTrackUsage` — single usage accumulates tokens and cost. File: `internal/tui/controller_test.go` (new)
- [x] 3.2 RED — Test: `TestTrackUsageMultiple` — multiple usages sum correctly. File: `internal/tui/controller_test.go`
- [x] 3.3 GREEN — Add fields `totalTokens int`, `contextWindow int`, `modelName string`, `modelPricing map[string]modelPrice` to Controller. File: `internal/tui/controller.go`
- [x] 3.4 GREEN — Add `TrackUsage(usage core.Usage)`, `Cost() float64`, `TotalTokens() int` methods with hardcoded pricing map. File: `internal/tui/controller.go`

## Phase 4: MCP Status

- [x] 4.1 RED — Test: `TestMCPManagerStatus` — connected/failed counts from mock clients. File: `internal/mcp/manager_test.go` (new)
- [x] 4.2 GREEN — Add `Status() (connected, failed int)` method to `MCPManager` — count entries in `clients` map. File: `internal/mcp/manager.go`

## Phase 5: App Integration

- [x] 5.1 RED — Test: `TestAppViewIncludesFooter` — View() output contains footer content. File: `internal/tui/app_test.go` (new)
- [x] 5.2 GREEN — Add `footer views.FooterModel` field to `App`. File: `internal/tui/app.go`
- [x] 5.3 GREEN — Update `NewAppWithTheme()` to initialize `FooterModel`. File: `internal/tui/app.go`
- [x] 5.4 GREEN — Update `View()` to append footer after input line (1 line, always visible). File: `internal/tui/app.go`
- [x] 5.5 GREEN — Update `streamDoneMsg` handler in `Update()` to call `ctrl.TrackUsage()` and sync footer via `rebuildViews()`. File: `internal/tui/app.go`
- [x] 5.6 GREEN — Update `rebuildViews()` to populate footer from controller state (tokens, cost, model, MCP status). File: `internal/tui/app.go`
