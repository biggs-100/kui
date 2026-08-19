# Proposal: Production TUI

## Intent

kui's TUI is a functional prototype compared to OpenCode's production TUI. The goal is to close the gap and build a production-quality interface that users will love.

## Current Gap (20 items)

1. No multi-line input
2. No slash command autocomplete
3. No command palette
4. No sidebar (tokens, cost, files)
5. No footer (status bar)
6. No diff rendering
7. No session management UI
8. No dialogs/modals
9. No which-key
10. No markdown rendering
11. No syntax highlighting in chat
12. No toast notifications
13. No clipboard integration
14. No mouse support
15. No terminal title
16. No thinking mode toggle
17. No model/agent cycling
18. No file attachments
19. No plugin system
20. No theme variety (only 2)

## Phased Approach

### Phase 1: Input Revolution (Foundation)
**Goal**: Modern input experience
- Multi-line textarea with cursor movement
- Word delete, undo/redo
- Clipboard paste
- Slash command autocomplete
- Command history (frecency)

**Files**: `internal/tui/views/input.go` (new)

### Phase 2: Status Display (Context)
**Goal**: Show what's happening
- Footer status bar (directory, model, tokens)
- Token count display
- Cost tracking
- MCP/LSP status indicators
- Terminal title update

**Files**: `internal/tui/views/footer.go` (new), `internal/tui/app.go` (modified)

### Phase 3: Session Management (Persistence UI)
**Goal**: Manage conversations
- Session list with metadata
- Session rename
- Session fork (branch from point)
- Session search
- Undo/redo within session

**Files**: `internal/tui/views/sessions.go` (new), `internal/tui/views/dialog.go` (new)

### Phase 4: Diff Rendering (Code Visibility)
**Goal**: See what changed
- Unified diff viewer
- File tree with change counts
- Hunk navigation
- Revert support
- Line numbers

**Files**: `internal/tui/views/diff.go` (new), `internal/tui/views/files.go` (new)

### Phase 5: Command Palette (Power User)
**Goal**: Fast access to everything
- Fuzzy search across commands
- Which-key overlay
- Keyboard shortcut discoverability
- Model/agent cycling
- Thinking mode toggle

**Files**: `internal/tui/views/palette.go` (new), `internal/tui/views/whichkey.go` (new)

### Phase 6: Polish & Plugins (Production)
**Goal**: Production quality
- Toast notifications
- Markdown rendering in chat
- Syntax highlighting
- Plugin system
- Mouse support
- Theme variety (port 10+ themes from OpenCode)

**Files**: Various

## Success Criteria

1. Multi-line input with autocomplete works
2. Footer shows real-time token/cost info
3. Sessions can be listed, renamed, forked
4. Diffs are visible and navigable
5. Command palette provides fuzzy search
6. 10+ themes available
7. All features feel responsive and polished
