# Archive: polish-plugins

## Summary

Added visual polish to kui's TUI: toast notifications, markdown-to-lipgloss rendering with Chroma syntax highlighting, and 12 new themes with cycling commands. Implemented via strict TDD across 7 phases (toast, toast integration, markdown rendering, syntax highlighting, theme variety, app integration, dependencies). All 19 tasks complete, all tests PASS.

## Implementation

### Files Created
- `internal/tui/toast/model.go` — ToastModel with push/tick/dismiss lifecycle
- `internal/tui/toast/model_test.go` — Unit tests for toast lifecycle
- `internal/tui/markdown/renderer.go` — Regex-based markdown→lipgloss renderer (~200 lines)
- `internal/tui/markdown/renderer_test.go` — Golden tests for 7 markdown patterns
- `internal/tui/markdown/highlight.go` — Chroma-based syntax highlighting with theme-mapped style
- `internal/tui/markdown/highlight_test.go` — Tests for syntax highlighting output
- `themes/catppuccin-mocha.json` — Catppuccin Mocha theme
- `themes/dracula.json` — Dracula theme
- `themes/gruvbox-dark.json` — Gruvbox Dark theme
- `themes/nord.json` — Nord theme
- `themes/tokyonight.json` — Tokyo Night theme
- `themes/one-dark.json` — One Dark theme
- `themes/rose-pine.json` — Rosé Pine theme
- `themes/everforest.json` — Everforest theme
- `themes/kanagawa.json` — Kanagawa theme
- `themes/monokai.json` — Monokai theme
- `themes/snazzy.json` — Snazzy theme
- `themes/modus-vivendi.json` — Modus Vivendi theme

### Files Modified
- `internal/tui/views/chat.go` — Assistant messages rendered through `markdown.Render()`
- `internal/tui/app.go` — Toast model wired into Update/View; theme cycling with next/prev
- `internal/tui/theme/theme.go` — Added `ThemeNames()` helper
- `go.mod` — Added `github.com/alecthomas/chroma/v2` dependency

### Tests
- All tests PASS across all modified packages
- `go vet` clean

## Verification

- Requirements: All success criteria met
- Phases: 7/7 complete (toast, integration, markdown, highlighting, themes, app, deps)
- Tasks: 19/19 complete (per persisted `tasks.md` artifact)
- Themes: 14 total (2 original + 12 new) — exceeds 12+ requirement

### Warnings
- None

## Known Issues

- Open questions from design.md remain: markdown rendering scope (user vs assistant messages), toast default duration configurability, theme persistence across restarts. These are deferred design decisions, not implementation defects.

## Engram Traceability

| Artifact | Observation ID |
|----------|---------------|
| Proposal | #18521 |
| Tasks | #18522 |
| Apply Progress | #18523 |
| Verify Report | #18524 |
| Archive Report | (this artifact) |
