# Design: Polish Plugins

## Technical Approach

Extend kui's TUI with four visual polish features: toast notifications, markdown rendering, syntax highlighting, and theme variety. All changes are additive — no existing rendering logic is replaced, only wrapped or augmented. The markdown renderer intercepts assistant message content before `ChatModel.Render()` outputs it, applying lipgloss styling. Chroma integrates at the code-fence level only. Themes are plain JSON files using the existing `Theme` struct.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|-------------|-----------|
| Toast placement | Separate Bubbletea model composed in `App` | Overlay in chat view | Keeps toast lifecycle independent from chat; avoids complex z-index math in lipgloss |
| Markdown parser | Custom lightweight parser (regex-based) | Goldmark, glamour | Goldmark pulls heavy dependencies; glamour wraps chroma but adds complexity. A ~200-line regex parser handles the 7 patterns we need (headings, bold, italic, inline code, fences, lists, blockquotes) |
| Syntax highlighting | Chroma v2 with theme-mapped style | Tree-sitter, no highlighting | Chroma is the standard Go highlighter; Theme already has `Syntax*` fields mapping to Chroma `StyleEntry` slots |
| Theme porting | JSON files in `themes/` dir | Code-embedded themes | Follows existing pattern (`kui-default.json`, `solarized-osaka.json`); user-extensible |
| Theme cycling | `/theme next` + `/theme prev` commands | Keybinding | Consistent with existing `/theme <name>` pattern; discoverable via palette |

## Data Flow

### Toast Notifications

```
Controller event ──→ App.Update() ──→ toast.Push(msg, level, duration)
                                            │
                                     tea.Cmd timer
                                            │
                                     toast.TickMsg → auto-dismiss
                                            │
                                     App.View() ──→ toast.Render() appended below chat
```

### Markdown + Syntax Highlighting

```
Assistant message.Content
    │
    ▼
markdown.Render(content, styles)  ← new function
    │
    ├── Split into segments: text, heading, bold, italic, code-inline, code-fence, list, blockquote
    │
    ├── For code-fence segments:
    │     chromaLexier(content, language) → chroma.Style → lipgloss-styled tokens
    │
    └── Return []string (styled segments joined by newlines)
    │
    ▼
ChatModel.Render() uses rendered output instead of raw msg.Content
```

### Theme Cycling

```
/theme next ──→ theme.List() → find current index → index+1 % len
                    │
              theme.Load(name) → NewStyles(t) → a.styles = styles
                    │
              a.rebuildViews() + toast("theme: <name>")
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/toast/model.go` | Create | ToastModel: queue, push, tick, render overlay |
| `internal/tui/toast/model_test.go` | Create | Unit tests for toast lifecycle |
| `internal/tui/markdown/renderer.go` | Create | Markdown→lipgloss renderer (~200 lines) |
| `internal/tui/markdown/renderer_test.go` | Create | Golden tests for each markdown pattern |
| `internal/tui/markdown/highlight.go` | Create | Chroma integration: theme→Style mapping + lex |
| `internal/tui/markdown/highlight_test.go` | Create | Tests for syntax highlighting output |
| `internal/tui/views/chat.go` | Modify | `Render()` calls `markdown.Render()` for assistant messages |
| `internal/tui/views/chat_test.go` | Modify | Update golden files; add markdown rendering tests |
| `internal/tui/app.go` | Modify | Add `toast toast.Model` field; wire into Update/View |
| `internal/tui/commands.go` | Modify | Register `/theme next`, `/theme prev` |
| `internal/tui/theme/styles.go` | Modify | Add `Markdown*` styles (heading, bold, italic, code, list, blockquote) |
| `internal/tui/theme/theme.go` | Modify | Add `ThemeNames()` helper; store current theme name in App |
| `themes/*.json` | Create | 10+ theme files (Tokyo Night, Catppuccin, Gruvbox, Dracula, One Dark, Nord, Rosé Pine, Everforest, Kanagawa, Monokai, Snazzy, Modus Vivendi) |
| `go.mod` | Modify | Add `github.com/alecthomas/chroma/v2` |

## Interfaces / Contracts

### Toast

```go
// internal/tui/toast/model.go
type Level int
const (
    LevelInfo Level = iota
    LevelSuccess
    LevelWarn
    LevelError
)

type Toast struct {
    Text     string
    Level    Level
    Duration time.Duration
}

type Model struct {
    toasts []Toast
    styles *theme.Styles
}

func NewModel(styles *theme.Styles) Model
func (m *Model) Push(text string, level Level, duration time.Duration)
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd)
func (m Model) View() string  // returns empty string when no toasts
```

### Markdown Renderer

```go
// internal/tui/markdown/renderer.go
func Render(content string, styles *theme.Styles) string
```

Takes raw markdown, returns lipgloss-styled string. Unrecognized syntax passes through as plain text.

### Syntax Highlighter

```go
// internal/tui/markdown/highlight.go
func HighlightCode(code string, lang string, t *theme.Theme) string
```

Maps `Theme.Syntax*` fields to Chroma `chroma.Style` entries. Falls back to monochrome if language is unknown.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Toast push/dismiss lifecycle | Table-driven tests: push toast, simulate tick, verify removal |
| Unit | Markdown rendering per pattern | Golden tests for each: heading, bold, italic, code-inline, code-fence, list, blockquote |
| Unit | Syntax highlighting | Verify Chroma output contains ANSI codes for known languages (go, python, js) |
| Unit | Theme cycling | Verify index wrapping, list correctness |
| Integration | Chat render with markdown | `ChatModel.Render()` produces styled output for assistant messages containing markdown |
| Integration | Toast overlay in App.View() | Verify toast renders below chat content when active |
| E2E | Theme switch updates all views | Manual: `/theme catppuccin` → verify header, chat, footer all recolor |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. All changes are additive:
- New packages (`toast/`, `markdown/`) don't affect existing imports
- `chat.go` modification is a single line change (wrap `msg.Content` in `markdown.Render()`)
- Theme files are discovered via existing `theme.Discover()` mechanism
- `go.mod` addition is backward-compatible

Feature flag not needed — all features are safe to enable immediately.

## Open Questions

- [ ] Should markdown rendering apply to user messages too, or only assistant? (Proposal says assistant only — confirm)
- [ ] Toast default duration: 3 seconds for info, 5 for errors? Or configurable?
- [ ] Theme persistence: save last-used theme to `.kui/config.yaml` or reset to default on restart?
