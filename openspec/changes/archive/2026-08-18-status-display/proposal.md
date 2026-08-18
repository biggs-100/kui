# Proposal: Status Display

## Intent

kui has no footer, no token tracking, no cost display, and no MCP/LSP status. OpenCode shows tokens, cost, context percentage, and MCP/LSP status in a footer bar. This change adds a status footer to kui's TUI.

## Current Gap

| Feature | kui | OpenCode |
|---------|-----|----------|
| Footer/status bar | ❌ | ✅ |
| Token count | ❌ | ✅ (total + breakdown) |
| Cost tracking | ❌ | ✅ (per-session USD) |
| Context window % | ❌ | ✅ |
| MCP status | ❌ | ✅ (per-server dots) |
| LSP status | ❌ | ✅ (per-server dots) |
| Streaming indicator | ❌ | ✅ (implied) |

## Proposed Solution

### Phase 2A: Footer View
Add a `FooterModel` that renders at the bottom of the screen:
```
~/project | gpt-4 | 1,234 tokens (12%) | $0.05 | MCP: 2 connected
```

### Phase 2B: Token Tracking
Track tokens per session:
- Total tokens used
- Context window percentage
- Update after each provider response

### Phase 2C: Cost Tracking
Track cost per session:
- Calculate from token counts and model pricing
- Display in footer when > 0

### Phase 2D: MCP Status
Show MCP server status:
- Count of connected servers
- Status dots (green/red) per server

## Scope

### In Scope
1. FooterModel with lipgloss styling
2. Token count display
3. Context window percentage
4. Cost calculation and display
5. MCP server count with status
6. Theme integration (use existing StatusOK/Error/Warn colors)

### Out of Scope
1. LSP status (no LSP in kui yet)
2. Sidebar (too complex for this phase)
3. Per-server MCP details (just count)
4. Token breakdown (input/output/reasoning)

## Success Criteria

1. Footer renders at bottom of screen
2. Token count updates after each response
3. Context percentage calculates correctly
4. Cost displays when > 0
5. MCP server count shows connected/failed
6. Footer uses theme colors

## Risks

1. **Token counting accuracy**: Provider may not return exact counts
   - Mitigation: Use reported usage, estimate if unavailable
2. **Cost calculation**: Model pricing varies
   - Mitigation: Hardcode common models, allow override
3. **Performance**: Footer updates on every chunk
   - Mitigation: Debounce updates, render only on change
