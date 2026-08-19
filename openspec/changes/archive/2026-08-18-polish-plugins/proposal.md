# Proposal: Polish Plugins

## Intent

kui's TUI is functional but lacks the visual polish users expect from a modern terminal agent. Chat messages render as raw text, there's no feedback mechanism for async actions, code blocks have no syntax coloring, and only 2 themes exist. This change closes the most visible quality gaps.

## Current Gap

1. **No toast notifications** — errors, saves, and config reloads produce no user-visible feedback unless the user is watching the right view
2. **No markdown rendering** — assistant responses with headers, lists, bold, or code fences render as raw markdown syntax
3. **No syntax highlighting** — code blocks in assistant answers display as monochrome text despite `Theme` already defining `Syntax*` colors
4. **No theme variety** — only `kui-default` and `solarized-osaka` ship; no way to discover or switch themes from the UI

## Scope

### In Scope
- Toast notification system (non-blocking, auto-dismissing status messages)
- Markdown-to-lipgloss renderer for assistant messages (headers, bold, italic, lists, inline code, code fences)
- Syntax highlighting for fenced code blocks using Chroma
- 10+ theme JSON files ported from OpenCode's palette
- Theme cycling command (`/theme next` or palette shortcut)

### Out of Scope
- Plugin system (deferred to future work)
- Mouse support
- Clipboard integration
- File attachments

## Capabilities

### New Capabilities
- `toast-notifications`: Non-blocking, auto-dismissing feedback toasts rendered in the TUI
- `markdown-rendering`: Parse markdown in assistant messages and render via lipgloss
- `syntax-highlighting`: Chroma-based highlighting for fenced code blocks in chat
- `theme-variety`: 10+ bundled themes and a command to cycle/preview them

### Modified Capabilities
- `tui-chat`: Message rendering pipeline updated to support markdown + syntax highlighting

## Approach

1. **Toast system** — New `internal/tui/toast/model.go`: a `ToastModel` that queues `{text, level, duration}` messages, renders as a fixed overlay at the bottom of the chat region, auto-dismisses via `tea.Cmd` timer. Integrated into `app.go`'s `Update` loop.

2. **Markdown renderer** — New `internal/tui/markdown/renderer.go`: lightweight markdown parser that converts to lipgloss-styled strings. Handles `#`-headings, `**bold**`, `*italic*`, `` `code` ``, ` ``` fences ```, `-` lists, and blockquotes. Returns `[]tea.Model` segments for composable rendering.

3. **Syntax highlighting** — Integrate `github.com/alecthomas/chroma/v2` with `chroma/lexers` for language detection. Theme's `Syntax*` colors map to Chroma `StyleEntry` slots. Applied inside the markdown renderer for fenced code blocks.

4. **Theme variety** — Port 10+ themes from OpenCode's `themes/` directory (Tokyo Night, Catppuccin, Gruvbox, Dracula, One Dark, Nord, Rosé Pine, Everforest, Kanagawa, Monokai, Snazzy, Modus Vivendi). Each is a JSON file matching the existing `Theme` struct. Add `/theme cycle` command and `--theme` flag for preview.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/tui/toast/` | New | Toast model and rendering |
| `internal/tui/markdown/` | New | Markdown parser + renderer |
| `internal/tui/views/chat.go` | Modified | Render pipeline calls markdown renderer instead of raw `msg.Content` |
| `internal/tui/app.go` | Modified | Wire toast model into Update/View cycle |
| `internal/tui/commands.go` | Modified | Add `/theme` command |
| `themes/*.json` | New | 10+ theme files |
| `go.mod` | Modified | Add `chroma/v2` dependency |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Chroma adds significant binary size | Low | Chroma compiles to ~3MB; acceptable for a TUI binary |
| Markdown parser misses edge cases | Medium | Start with common patterns; fallback to raw text for unrecognized syntax |
| Theme porting introduces color inconsistencies | Low | Validate each theme JSON with a schema check before committing |
| Toast overlay overlaps chat during resize | Low | Toast renders at fixed bottom offset; test resize scenarios |

## Rollback Plan

- Toast: Remove `internal/tui/toast/` package, revert `app.go` wiring
- Markdown: Revert `chat.go` to raw `msg.Content` rendering
- Themes: Delete extra JSON files from `themes/`
- All changes are additive; no existing behavior is modified, only extended

## Dependencies

- `github.com/alecthomas/chroma/v2` — syntax highlighting (add to `go.mod`)
- Existing theme JSON schema (`internal/tui/theme/theme.go`) — no changes needed

## Success Criteria

- [ ] Toast appears for 3 seconds then auto-dismisses when config reloads
- [ ] Assistant messages render `#` headings as bold colored text
- [ ] Fenced code blocks display with per-token syntax coloring
- [ ] 12+ themes load from `themes/` directory
- [ ] `/theme cycle` switches to next theme and refreshes all views
- [ ] `go test ./...` passes with no regressions
