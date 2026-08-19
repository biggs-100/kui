# Tasks: Polish Plugins

## Review Workload Forecast

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

| Field | Value |
|-------|-------|
| Estimated changed lines | ~330 |
| Delivery strategy | auto-chain |

### Suggested Work Units

| Unit | Goal | PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|----|----------------------|-----------------|-------------------|
| 1 | Toast notifications | 1 | `go test ./internal/tui/toast/...` | N/A — pure model | `internal/tui/toast/`, `app.go` toast wiring |
| 2 | Markdown + syntax | 1 | `go test ./internal/tui/markdown/...` | N/A — pure renderer | `internal/tui/markdown/`, `chat.go` render call |
| 3 | Theme variety + cycling | 1 | `go test ./internal/tui/... -run TestTheme` | N/A — no runtime | `themes/*.json`, `commands.go` `/theme` |

## Phase 1: Toast Notifications

- [x] 1.1 RED — `internal/tui/toast/model_test.go`: table-driven `TestToastCreate` (push, verify state), `TestToastPush` (multiple queued), `TestToastDismiss` (tick removes expired)
- [x] 1.2 GREEN — `internal/tui/toast/model.go`: `Level` enum, `Toast` struct, `Model` with `Push()`, `Update()` (tickMsg), `View()` (styled overlay or empty)
- [x] 1.3 RED — `TestToastRender` in `model_test.go`: `View()` output includes styled text per Level (info/success/warn/error)

## Phase 2: Toast Integration

- [x] 2.1 RED — `internal/tui/app_test.go`: `TestAppToast` — push + tick, verify toast in `View()`
- [x] 2.2 GREEN — Add `toast toast.Model` field to `App` in `app.go`; wire `toast.Update()` and `toast.View()`

## Phase 3: Markdown Rendering

- [x] 3.1 RED — `internal/tui/markdown/renderer_test.go`: `TestRenderHeading` (`# H1` styled), `TestRenderBold` (`**bold**` styled), `TestRenderInlineCode` (`` `code` `` styled), `TestRenderList` (`- item` styled)
- [x] 3.2 GREEN — `internal/tui/markdown/renderer.go`: `func Render(content string, styles *theme.Styles) string` — regex parser for headings, bold, italic, inline code, fences, lists, blockquotes
- [x] 3.3 RED — `TestRenderFence` in `renderer_test.go`: fenced block renders with syntax-styled tokens for known languages

## Phase 4: Syntax Highlighting

- [x] 4.1 RED — `internal/tui/markdown/highlight_test.go`: `TestHighlightGo` (Go code → ANSI output), `TestHighlightUnknown` (unknown lang → monochrome fallback)
- [x] 4.2 GREEN — `internal/tui/markdown/highlight.go`: `func HighlightCode(code string, lang string, t *theme.Theme) string` — maps `Theme.Syntax*` to Chroma `Style`, fallback for unknown languages

## Phase 5: Theme Variety

- [x] 5.1 Create 12 theme JSON files in `themes/`: `catppuccin`, `dracula`, `gruvbox`, `nord`, `tokyonight`, `one-dark`, `rose-pine`, `everforest`, `kanagawa`, `monokai`, `snazzy`, `modus-vivendi` — each matching `Theme` struct
- [x] 5.2 RED — `internal/tui/app_test.go`: `TestThemeCycling` — `/theme next` wraps, `/theme prev` wraps, name updates
- [x] 5.3 GREEN — Extend `handleThemeCommand` in `commands.go`: `next`/`prev` subcommands using `theme.List()`, index wrap, `theme.Load()`, update styles

## Phase 6: App Integration

- [x] 6.1 RED — `internal/tui/views/chat_test.go`: `TestChatMarkdownRender` — assistant `# Heading` renders styled (not raw `#`)
- [x] 6.2 GREEN — Modify `ChatModel.Render()` in `chat.go`: wrap assistant `msg.Content` in `markdown.Render()`
- [x] 6.3 GREEN — Wire toast in `app.go`: `toast.Push()` on config reload success, error, and theme switch

## Phase 7: Dependencies

- [x] 7.1 `go get github.com/alecthomas/chroma/v2` + verify `go.mod`/`go.sum`
- [x] 7.2 `go test ./...` — zero regressions, all new tests pass
- [x] 7.3 Add `ThemeNames() []string` to `theme.go` if not present (used by `/theme next|prev`)
