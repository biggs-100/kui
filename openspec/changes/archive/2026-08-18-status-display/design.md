# Design: Status Display

## Technical Approach

Add a `FooterModel` view to kui's TUI that renders session metadata at the bottom of the screen. Token tracking accumulates `core.Usage` data from streaming responses. Cost is calculated from a hardcoded model-pricing map. MCP status reads connected server count from the existing `MCPManager`. The footer reuses existing theme colors (`StatusOK`, `StatusError`, `StatusWarn`, `BGStatusline`) and follows the same view pattern as `HeaderModel` and `ToolModel`.

## Architecture Decisions

| Decision | Option A | Option B | Tradeoff | Decision |
|----------|----------|----------|----------|----------|
| Footer data source | Setter methods on FooterModel | Struct fields set via `rebuildViews()` | Setter = mutable, more Bubbletea idiomatic; struct fields = immutable, simpler | **Setter methods** — matches `ToolModel.AppendCall` pattern already in codebase |
| Token state location | In Controller | In App | Controller owns session state; App is pure composition | **Controller** — aligns with existing `messages`, `sessionStore` pattern |
| Cost calculation | In FooterModel | In Controller | FooterModel is a view; Controller owns domain logic | **Controller** — view renders data, doesn't compute |
| MCP status source | MCPManager.Status() | Config + clients map | Manager already tracks connected clients; Config doesn't reflect runtime state | **MCPManager.Status()** — needs new method, but correct abstraction |

## Data Flow

```
Controller (owns state)          FooterModel (renders)
┌─────────────────────┐         ┌──────────────────────┐
│ totalTokens int     │──set──→│ tokens int            │
│ contextWindow int   │──set──→│ contextMax int        │
│ cost float64        │──set──→│ cost float6          │
│ modelName string    │──set──→│ model string          │
│ directory string    │──set──→│ dir string            │
│ mcpStatus MCStatus  │──set──→│ mcpConnected int      │
│                     │         │ mcpFailed int         │
└─────────────────────┘         └──────────────────────┘
         ▲                              │
         │ streamDoneMsg               │ Render()
         │ (contains Usage)             ▼
    App.Update()                    lipgloss string
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/views/footer.go` | Create | FooterModel with setter methods and Render() |
| `internal/tui/views/footer_test.go` | Create | Unit tests for footer rendering and state |
| `internal/tui/theme/styles.go` | Modify | Add footer styles (StatusLine, StatusOK/Error/Warn BG) |
| `internal/tui/controller.go` | Modify | Add token/cost tracking fields, setter methods, cost calculation |
| `internal/tui/app.go` | Modify | Add footer field, wire in View(), pass usage data from streamDoneMsg |
| `internal/mcp/manager.go` | Modify | Add `Status() (connected, failed int)` method |

## Component Design

### FooterModel

```go
// FooterModel renders the status footer bar.
type FooterModel struct {
    dir          string
    model        string
    tokens       int
    contextMax   int
    cost         float64
    mcpConnected int
    mcpFailed    int
    styles       *theme.Styles
}

func NewFooterModel(styles *theme.Styles) FooterModel
func (m *FooterModel) SetDirectory(dir string)
func (m *FooterModel) SetModel(model string)
func (m *FooterModel) SetTokens(tokens, contextMax int)
func (m *FooterModel) SetCost(cost float64)
func (m *FooterModel) SetMCPStatus(connected, failed int)
func (m FooterModel) Render() string
```

### Controller Token Tracking

Add to Controller struct:

```go
totalTokens   int
contextWindow int  // default 128000
modelPricing  map[string]modelPrice

type modelPrice struct {
    inputPerToken  float64
    outputPerToken float64
}
```

Methods:
- `TrackUsage(usage core.Usage)` — accumulates tokens, recalculates cost
- `Cost() float64` — returns current session cost
- `TotalTokens() int` — returns accumulated tokens
- `ModelPricing()` — hardcoded map for gpt-4, gpt-4o, claude-3.5-sonnet

### MCP Status

Add to `MCPManager`:

```go
func (m *MCPManager) Status() (connected, failed int)
```

Returns count of connected vs failed clients from the internal `clients` map.

### App.View() Layout

```
header           (1 line)
chat             (remaining)
tool             (≤ height/4)
input            (1 line)
footer           (1 line)  ← NEW
```

Footer always renders. When no data yet, shows: `~/dir | model | — tokens | — cost | MCP: 0`

## Theme Styles

Add to `theme.Styles`:

```go
StatusLine   lipgloss.Style  // BGStatusline background, FG text
StatusOK     lipgloss.Style  // green dot for MCP connected
StatusError  lipgloss.Style  // red dot for MCP failed
StatusWarn   lipgloss.Style  // yellow for warnings
```

These use the existing `Theme.StatusOK`, `Theme.StatusError`, `Theme.StatusWarn`, `Theme.BGStatusline` fields that are already defined but unused.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | FooterModel.Render() with various states | Table-driven tests: empty state, partial data, full data, zero cost, high context % |
| Unit | Controller.TrackUsage() accumulation | Table-driven: single usage, multiple usages, zero usage |
| Unit | Controller cost calculation | Table-driven: known model pricing, unknown model (zero cost) |
| Unit | MCPManager.Status() | Mock client factory, verify connected/failed counts |
| Unit | Theme footer styles | Verify styles use correct theme colors |
| Integration | App.View() includes footer | Verify footer appears in rendered output |

## Migration / Rollout

No migration required. All new code. Token tracking defaults to zero; footer renders gracefully with no data.

## Open Questions

- [ ] Should context window size be configurable per model or hardcoded?
- [ ] Should the footer hide when terminal height < 10 lines?
