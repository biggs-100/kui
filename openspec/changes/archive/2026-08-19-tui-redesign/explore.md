# Exploration: TUI Redesign for OpenCode-style UI

## Goal
Make kui's TUI visually similar to OpenCode's TUI using Go/Bubble Tea.

## Current kui TUI Architecture

### Files:
- `internal/tui/app.go` — Main app, layout, key handling
- `internal/tui/controller.go` — Profile switching, prompt submission
- `internal/tui/views/header.go` — Header with profile info
- `internal/tui/views/chat.go` — Chat message display
- `internal/tui/views/footer.go` — Footer with tokens/cost/MCP/LSP
- `internal/tui/views/tool.go` — Tool call display
- `internal/tui/views/diff.go` — Diff view
- `internal/tui/commands.go` — Command registry
- `internal/tui/input.go` — Input model with history
- `themes/` — Theme system with lipgloss

### Layout (current):
```
┌─────────────────────────────────────┐
│ Header: profile | model             │
├─────────────────────────────────────┤
│ Chat messages                       │
│                                     │
│                                     │
├─────────────────────────────────────┤
│ Tool output (when active)           │
├─────────────────────────────────────┤
│ Footer: tokens | $cost | MCP | LSP │
├─────────────────────────────────────┤
│ Input: type here...                 │
└─────────────────────────────────────┘
```

## OpenCode TUI Architecture

### Files (reference):
- `packages/tui/src/routes/home.tsx` — Home layout
- `packages/tui/src/routes/session/footer.tsx` — Footer
- `packages/tui/src/component/logo.tsx` — Logo
- `packages/tui/src/component/prompt/` — Prompt input
- `packages/tui/src/theme/` — Theme system

### Layout (OpenCode):
```
┌─────────────────────────────────────┐
│                                     │
│                                     │
│           ┌─────────────┐           │
│           │   LOGO      │           │
│           │   ASCII     │           │
│           └─────────────┘           │
│                                     │
│      ┌─────────────────────┐        │
│      │ > Type here...      │        │
│      └─────────────────────┘        │
│                                     │
│                                     │
├─────────────────────────────────────┤
│ dir • LSP • MCP • /status           │
└─────────────────────────────────────┘
```

## Key Visual Differences

| Aspect | kui (current) | OpenCode |
|--------|---------------|----------|
| Home layout | Chat-first | Centered logo + prompt |
| Header | Always visible | Hidden on home |
| Footer | tokens/cost/MCP/LSP | dir • LSP • MCP |
| Colors | Theme-dependent | Minimal grays |
| Logo | None | ASCII art |
| Prompt | Bottom input box | Centered with border |
| Status | In footer | /status command |

## What Can Be Replicate with Bubble Tea

1. **Centered home layout** — `lipgloss.Place()` for centering
2. **ASCII logo** — Simple text rendering
3. **Minimal footer** — `lipgloss.JoinHorizontal()` with styles
4. **Bordered prompt** — `lipgloss.Border(lipgloss.RoundedBorder())`
5. **Gray color scheme** — `lipgloss.Color()` with hex values
6. **Model selector dialog** — `list.Model` from bubbles

## What Requires Major Rewrites

1. **Route system** — kui has no route concept (home vs session)
2. **SolidJS components** — Can't port directly, must reimplement
3. **Plugin system** — OpenCode has extensive plugin slots
4. **Theme variables** — Different naming/structure

## Recommended Approach

### Phase 1: Home Screen Redesign
- Add route system (home vs session)
- Create centered home layout with logo
- Redesign footer to match OpenCode style
- Add bordered prompt input

### Phase 2: Theme & Colors
- Create new "opencode" theme
- Minimal gray color palette
- Subtle borders and separators

### Phase 3: Dialogs & Interactions
- Model selection dialog (list with preview)
- Session list dialog
- Command palette improvements

### Phase 4: Polish
- Transitions between home and session
- Status indicators (LSP, MCP)
- Keyboard shortcuts matching OpenCode

## Estimated Effort
- Phase 1: 2-3 days
- Phase 2: 1 day
- Phase 3: 2-3 days
- Phase 4: 1-2 days
- **Total: 6-9 days**

## Key Learnings

1. OpenCode uses SolidJS with @opentui/solid, not Bubble Tea
2. The visual design can be replicated with lipgloss styles
3. A route system is needed to switch between home and session views
4. The footer should show directory and status, not token counts
5. A centered logo + prompt layout is the signature OpenCode look
