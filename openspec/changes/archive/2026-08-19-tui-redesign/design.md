# Design: TUI Redesign — OpenCode-style UI

## Architecture Overview

The redesign adds a route system to switch between home and session views, creates a new home screen component, and modifies the footer for different contexts.

## Current Architecture

```
App
├── header (HeaderModel)
├── chat (ChatModel)
├── tool (ToolModel)
├── footer (FooterModel)
├── diff (DiffModel)
├── input (InputModel)
└── controller (Controller)
```

## Proposed Architecture

```
App
├── route: home | session
├── home
│   ├── logo (LogoModel) [NEW]
│   ├── prompt (HomePromptModel) [NEW]
│   └── footer (HomeFooterModel) [NEW]
├── session
│   ├── header (HeaderModel)
│   ├── chat (ChatModel)
│   ├── tool (ToolModel)
│   ├── footer (SessionFooterModel) [MODIFIED]
│   └── input (InputModel)
├── overlay
│   ├── commandPalette
│   └── sessionList
└── controller (Controller)
```

## Key Design Decisions

### Decision 1: Route State in App

**Choice:** Add `route` field to App struct (home | session)

**Rationale:**
- Simple state machine, no complex router needed
- Route determines which view components render
- Preserves existing session view code unchanged

**Alternatives considered:**
- Separate HomeApp/SessionApp structs — rejected: too much duplication
- External router library — rejected: overkill for 2 routes

### Decision 2: Home Screen Components

**Choice:** Create new files for home-specific components

**New files:**
- `internal/tui/views/home.go` — HomeView model (logo + prompt + footer)
- `internal/tui/views/logo.go` — LogoModel (ASCII art rendering)
- `internal/tui/views/home_footer.go` — HomeFooterModel (minimal footer)

**Rationale:**
- Keeps home logic isolated from session logic
- Easy to toggle between views
- Testable independently

### Decision 3: Footer Strategy

**Choice:** Single FooterModel with mode parameter

```go
type FooterMode int
const (
    FooterModeHome FooterMode = iota
    FooterModeSession
)
```

**Rationale:**
- Reuses existing FooterModel code
- Mode determines which info to show
- Minimal code duplication

### Decision 4: Theme Extension

**Choice:** Add "opencode" theme to existing theme system

**New file:** `internal/tui/theme/opencode.go`

```go
var OpenCode = Theme{
    Name:    "opencode",
    BG:      lipgloss.Color("#1a1a1a"),
    Text:    lipgloss.Color("#e0e0e0"),
    Muted:   lipgloss.Color("#808080"),
    Accent:  lipgloss.Color("#569cd6"),
    Border:  lipgloss.Color("#333333"),
    Success: lipgloss.Color("#4ec9b0"),
    Error:   lipgloss.Color("#f44747"),
}
```

**Rationale:**
- Follows existing theme pattern
- Easy to select via `/theme opencode`
- Can be refined later

### Decision 5: Transition Strategy

**Choice:** Instant transition (no animation)

**Rationale:**
- Matches OpenCode behavior
- Simple to implement
- Animation can be added in Phase 2

## Data Flow

### Home → Session Transition

```
1. User types prompt on home screen
2. User presses Enter
3. App.handleCommand() detects empty command
4. App sets route = session
5. App calls ctrl.SubmitPrompt()
6. Session view renders with chat + tool
```

### Session → Home (Future)

```
1. User presses Ctrl+N (new session)
2. App sets route = home
3. App clears chat history
4. Home view renders
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/app.go` | MODIFY | Add route state, switch view rendering |
| `internal/tui/views/home.go` | CREATE | HomeView model |
| `internal/tui/views/logo.go` | CREATE | LogoModel |
| `internal/tui/views/home_footer.go` | CREATE | HomeFooterModel |
| `internal/tui/views/footer.go` | MODIFY | Add FooterMode parameter |
| `internal/tui/theme/opencode.go` | CREATE | OpenCode theme |
| `internal/tui/theme/theme.go` | MODIFY | Register opencode theme |
| `internal/tui/commands.go` | MODIFY | Add /theme to home palette |

## Test Strategy

### Unit Tests
- `home_test.go` — HomeView rendering, centering logic
- `logo_test.go` — Logo rendering, color application
- `home_footer_test.go` — Footer mode switching
- `footer_test.go` — Existing tests still pass

### Integration Tests
- `app_test.go` — Route switching, view transitions
- `theme_test.go` — OpenCode theme loads correctly

### Manual Testing
- Visual comparison with OpenCode screenshots
- Resize behavior on home and session views
- Keyboard shortcuts on home screen

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing TUI | High | Keep old code paths, feature flag |
| Route state bugs | Medium | Comprehensive unit tests |
| Theme inconsistency | Low | Manual visual testing |

## Rollback Plan

If redesign fails:
1. Remove `route` field from App
2. Remove home.go, logo.go, home_footer.go
3. Restore original view() method
4. Remove opencode theme

## Implementation Order

1. **Theme** — Create opencode theme (low risk, high value)
2. **Logo** — Create LogoModel (isolated, testable)
3. **Home Footer** — Create HomeFooterModel (isolated)
4. **Home View** — Compose home screen (depends on 2, 3)
5. **Route System** — Add route to App (depends on 4)
6. **Footer Mode** — Modify existing footer (backward compatible)
7. **Integration** — Wire everything together
8. **Polish** — Visual tweaks, edge cases

## Key Learnings

1. The existing theme system makes adding new themes straightforward
2. Bubble Tea's model composition works well for view switching
3. Footer can be extended with a mode parameter without breaking changes
4. Home screen is isolated from session logic, reducing risk
5. Lipgloss's Place() function handles centering elegantly
