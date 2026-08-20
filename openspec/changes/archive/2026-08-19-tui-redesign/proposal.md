# Proposal: TUI Redesign — OpenCode-style UI

## Intent

Redesign kui's TUI to visually match OpenCode's terminal interface. The goal is a modern, minimal UI centered on the home screen with a clean footer, bordered prompt, and gray color palette.

## Scope

### In Scope (First Slice)
1. **Home screen layout** — Centered logo + prompt + footer (like OpenCode)
2. **Route system** — Switch between home view and session/chat view
3. **Footer redesign** — Show `directory • LSP • MCP • /status` instead of tokens/cost
4. **Bordered prompt** — Rounded border input on home screen
5. **New theme** — "opencode" theme with minimal gray palette
6. **ASCII logo** — Generic "kui" logo (custom later)

### Out of Scope (Future)
- Model selection dialog (Ctrl+L) — Phase 2
- Session list dialog — Phase 2
- Animated transitions — Phase 2
- Plugin system — Not planned
- Custom ASCII art — User will decide later

## Target Users

- kui users who want a modern, clean TUI experience
- Users familiar with OpenCode's interface

## Affected Areas

| Area | Change |
|------|--------|
| `internal/tui/app.go` | Add route system, home view |
| `internal/tui/views/footer.go` | Redesign footer layout |
| `internal/tui/views/home.go` | New file: centered home view |
| `internal/tui/views/logo.go` | New file: ASCII logo |
| `themes/opencode.go` | New theme file |

## Current State Gap

kui's current TUI has a traditional layout:
- Header always visible
- Chat messages take full screen
- Footer shows tokens/cost/MCP/LSP
- No home screen — jumps directly to chat

OpenCode's TUI has a modern layout:
- Home screen with centered logo + prompt
- Header hidden on home
- Footer shows directory + status indicators
- Bordered prompt input

## Success Criteria

1. `kui` launches to a centered home screen with logo + prompt
2. Footer shows `directory • LSP • MCP • /status`
3. Pressing Enter sends prompt and transitions to chat view
4. Color scheme matches OpenCode's minimal gray palette
5. All existing functionality (commands, sessions, themes) still works

## Risks

| Risk | Mitigation |
|------|------------|
| Breaking existing TUI | Keep old views as fallback |
| Route system complexity | Simple state machine, not full router |
| Performance | Lipgloss is fast, no concerns |

## Rollback Plan

If redesign fails:
1. Remove `internal/tui/views/home.go`
2. Restore original `app.go` layout
3. Revert theme changes

## Business Rules

- Must maintain backward compatibility with existing commands
- Must work with all providers (openai, opencode, opencode-go)
- Must preserve TDD — all changes must have tests

## Decision Gaps

- [ ] Custom ASCII logo — deferred to user
- [ ] Model selector dialog — Phase 2
- [ ] Session list dialog — Phase 2
