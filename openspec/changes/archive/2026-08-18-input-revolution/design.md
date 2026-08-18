# Design: Input Revolution

## Technical Approach

Replace `App.input string` with `textarea.Model` from `charmbracelet/bubbles` (new dependency). Wrap textarea in a composite `InputModel` that adds history navigation and slash-command autocomplete. The App delegates key events to InputModel, which owns all input state. History persists to JSONL under `~/.config/kui/history.jsonl`. Autocomplete renders as an inline popup above the textarea when the input starts with `/`.

## Architecture Decisions

### Decision: Input widget — textarea.Model vs custom editor

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `bubbles/textarea.Model` | New dependency; rich feature set (cursor, undo, paste, word nav); keymap conflicts to resolve | **Chosen** — mature, well-tested, handles bracketed paste natively |
| Custom `contenteditable`-style editor | Full control; high implementation cost; reinvents cursor/selection/undo | Rejected — too much work for Phase 1A |
| `bubbles/textinput.Model` | Already available; single-line only; no word wrap | Rejected — proposal requires multi-line |

### Decision: History storage format

| Option | Tradeoff | Decision |
|--------|----------|----------|
| JSONL (one entry per line) | Append-only, no index corruption on crash, simple trim; no fast random access | **Chosen** — matches proposal spec, simple, inspectable |
| SQLite | Fast queries; new dependency; overkill for 50 entries | Rejected |
| Single JSON array | Simple; rewrite entire file on each save; corruption risk on crash | Rejected |

### Decision: Autocomplete rendering

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Inline popup (below/above textarea) | Simple; composable with existing layout; no overlay complexity | **Chosen** — fits kui's region-based layout |
| Lipgloss overlay/float | Visually rich; requires ANSI layering; breaks simple string concat layout | Deferred — Phase 2 |

### Decision: Keybinding conflict resolution

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Intercept before textarea.Update | Clean separation; app-level keys never reach textarea | **Chosen** — Tab, Ctrl+C, Enter-before-submit handled at App level |
| Modify textarea.KeyMap | Fewer intercepts; risk of breaking textarea internals | Rejected — fragile, harder to reason about |

## Data Flow

```
tea.KeyMsg
    │
    ▼
App.Update()
    │
    ├─ tea.WindowSizeMsg ──▶ textarea.SetWidth/SetHeight
    │
    ├─ Key interception layer
    │   ├─ Tab/Shift+Tab ──▶ SwitchProfile (never reaches textarea)
    │   ├─ Ctrl+C ──▶ SaveSession + tea.Quit
    │   ├─ Enter ──▶ if autocomplete active: select item
    │   │           else if input starts with /: handleCommand
    │   │           else: SubmitPrompt (clear textarea)
    │   └─ Escape ──▶ dismiss autocomplete
    │
    ├─ textarea.Update(msg) ──▶ textarea handles all other keys
    │   (arrows, home/end, ctrl+z/y, backspace, runes, etc.)
    │
    ├─ Autocomplete state check
    │   if input.TextValue() starts with "/" ──▶ compute matches
    │   if matches changed ──▶ update autocomplete popup
    │
    └─ History navigation
        Up/Down at cursor boundary ──▶ navigate history (1B)

View:
    header + chat + [autocomplete popup?] + textarea.View()
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `go.mod` / `go.sum` | Modify | Add `github.com/charmbracelet/bubbles` dependency |
| `internal/tui/app.go` | Modify | Replace `input string` with `textarea.Model`; rewrite `handleKey` to intercept + delegate; update `View()` |
| `internal/tui/input.go` | Create | `InputModel` wrapper: owns textarea.Model, history state, autocomplete state; exposes `Submit()`, `HistoryUp/Down()`, `Update()`, `View()` |
| `internal/tui/history.go` | Create | `InputHistory` — JSONL read/write, dedup, 50-entry limit, `~/.config/kui/history.jsonl` |
| `internal/tui/autocomplete.go` | Create | `AutocompleteModel` — command list, fuzzy match, selection index, `Update()`, `View()` |
| `internal/tui/app_test.go` | Modify | Update tests: `input` field replaced by `InputModel()` accessor; new tests for history, autocomplete |
| `internal/tui/history_test.go` | Create | Unit tests: load/save/dedup/trim, empty file, malformed JSONL |
| `internal/tui/autocomplete_test.go` | Create | Unit tests: match filtering, selection navigation, escape dismissal |

## Interfaces / Contracts

```go
// internal/tui/input.go

// InputModel wraps textarea.Model and adds history + autocomplete.
type InputModel struct {
    ta           textarea.Model
    history      InputHistory
    autocomplete AutocompleteModel
}

func NewInputModel(configDir string) InputModel

// Update delegates to textarea and checks for history/autocomplete state changes.
func (m *InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd)

// View returns the textarea rendered output.
func (m *InputModel) View() string

// Value returns the current text content.
func (m *InputModel) Value() string

// SetValue sets the textarea content (for history recall).
func (m *InputModel) SetValue(s string)

// Submit clears the textarea and returns the submitted text.
func (m *InputModel) Submit() string

// Focus gives the textarea focus.
func (m *InputModel) Focus() tea.Cmd
```

```go
// internal/tui/history.go

// InputHistory manages prompt history with JSONL persistence.
type InputHistory struct {
    entries []string
    index   int  // -1 = not browsing
    path    string
}

func NewInputHistory(configDir string) InputHistory
func (h *InputHistory) Load() error
func (h *InputHistory) Save(entry string) error  // dedup + trim to 50
func (h *InputHistory) Up(current string) string  // navigate back
func (h *InputHistory) Down() string               // navigate forward
func (h *InputHistory) Reset()                     // reset index to -1
```

```go
// internal/tui/autocomplete.go

// AutocompleteModel manages slash-command completion.
type AutocompleteModel struct {
    commands []string
    filtered []string
    index    int
    active   bool
}

var defaultCommands = []string{
    "/reload", "/sessions", "/resume", "/quit",
    "/help", "/theme", "/status", "/clear",
}

func NewAutocompleteModel() AutocompleteModel
func (a *AutocompleteModel) Activate(input string)  // filter + show
func (a *AutocompleteModel) Deactivate()
func (a *AutocompleteModel) Update(msg tea.KeyMsg) (selected string, handled bool)
func (a *AutocompleteModel) View() string           // rendered popup lines
func (a *AutocompleteModel) IsActive() bool
```

## Integration Points

**App.handleKey rewrite** — The current handler has a flat switch on `msg.Type`. The new handler:

1. Intercept Tab/Shift+Tab → profile cycle (unchanged)
2. Intercept Ctrl+C → quit (unchanged)
3. Intercept Escape → dismiss autocomplete
4. If autocomplete active: route Up/Down/Enter/Tab to autocomplete
5. Otherwise: delegate to `InputModel.Update(msg)` which routes to textarea
6. After update: check if input starts with `/` → activate autocomplete
7. Enter (no autocomplete): submit prompt, record history, clear textarea

**App.View rewrite** — Replace `"> " + a.input` with:

```go
inputLine := a.input.View()
if a.input.autocomplete.IsActive() {
    inputLine = a.input.autocomplete.View() + "\n" + inputLine
}
```

**Config dir**: `Wiring.ConfigRoot` passed to `NewInputModel(configDir)` in `run.go`.

**WindowSizeMsg**: Forward to `textarea.SetWidth(available)` so textarea wraps correctly.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `InputHistory` — load, save, dedup, trim, empty file, malformed JSONL | `go test` with `t.TempDir()` |
| Unit | `AutocompleteModel` — filter, navigate, dismiss, empty input | Table-driven tests |
| Unit | `InputModel` — value get/set, submit clears, history navigation | Direct `Update()` calls with `tea.KeyMsg` |
| Unit | App — Enter submits, Tab cycles, Ctrl+C quits (existing) | Update existing `app_test.go` with new model |
| Integration | Full input flow: type, history browse, autocomplete select, submit | `teatest.NewTestModel()` |
| Integration | History persistence across app restart | Write → reload → verify |
| E2E | Multi-line input with Shift+Enter, cursor movement, undo | Manual verification |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. The `textarea.Model` is a drop-in replacement for the string field. History file (`~/.config/kui/history.jsonl`) is created on first use. Existing kui installs have no history data — starts empty.

## Open Questions

- [ ] Should autocomplete popup use `lipgloss` styling (borders, colors) or plain text? (Design assumes styled, matching existing theme.)
- [ ] History file encoding: UTF-8 only, or handle binary content? (Design assumes UTF-8 text entries only.)
- [ ] Should Shift+Enter create a newline (multi-line) or submit? (Design assumes newline per proposal; confirm UX.)
