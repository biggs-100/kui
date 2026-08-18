# Tasks: Input Revolution

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 650–750 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 (stacked-to-main) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | InputModel + History foundation | PR 1 | `go test ./internal/tui/ -run TestInput\|TestHistory` | N/A — unit tests only | `internal/tui/input.go`, `internal/tui/history.go` and their tests |
| 2 | Autocomplete + Paste + App integration | PR 2 | `go test ./internal/tui/ -run TestAutocomplete\|TestPaste\|TestApp` | N/A — unit tests only | `internal/tui/autocomplete.go`, modified `app.go` and their tests |

## Phase 1A: InputModel Core

- [x] 1.1 RED — `TestInputModelCreate`: creates with placeholder string — `internal/tui/input_test.go`
- [x] 1.2 GREEN — Implement `InputModel` struct wrapping `textarea.Model` with `NewInputModel()` — `internal/tui/input.go`
- [x] 1.3 RED — `TestInputValue`: `Value()` returns current text — `internal/tui/input_test.go`
- [x] 1.4 RED — `TestInputClear`: `Submit()` clears textarea and returns submitted text — `internal/tui/input_test.go`
- [x] 1.5 RED — `TestInputCursorMove`: left/right/home/end keys move cursor — `internal/tui/input_test.go`
- [x] 1.6 RED — `TestInputWordNav`: Ctrl+Left/Right jumps by word — `internal/tui/input_test.go`
- [x] 1.7 RED — `TestInputUndoRedo`: Ctrl+Z/Y undo/redo — `internal/tui/input_test.go`
- [x] 1.8 GREEN — Wire `textarea.Model` into `InputModel`: `Update()`, `View()`, `Focus()`, `Blur()`, `SetValue()` — `internal/tui/input.go`

## Phase 1B: Input History

- [x] 2.1 RED — `TestHistoryCreate`: empty history loads cleanly — `internal/tui/history_test.go`
- [x] 2.2 GREEN — Implement `InputHistory` struct with JSONL persistence, `NewInputHistory()`, `Load()`, `Save()` — `internal/tui/history.go`
- [x] 2.3 RED — `TestHistoryAppend`: `Save()` adds entry, deduplicates consecutive — `internal/tui/history_test.go`
- [x] 2.4 RED — `TestHistoryNav`: `Up()`/`Down()` navigate entries — `internal/tui/history_test.go`
- [x] 2.5 RED — `TestHistoryLimit`: trims to 50 entries on save — `internal/tui/history_test.go`
- [x] 2.6 GREEN — Integrate history into `InputModel`: up/down arrows at cursor boundary — `internal/tui/input.go`

## Phase 1C: Slash Command Autocomplete

- [x] 3.1 RED — `TestAutocompleteCreate`: creates with default command list — `internal/tui/autocomplete_test.go`
- [x] 3.2 GREEN — Implement `AutocompleteModel` struct: `NewAutocompleteModel()`, `Activate()`, `Deactivate()`, `IsActive()` — `internal/tui/autocomplete.go`
- [x] 3.3 RED — `TestAutocompleteFilter`: `Activate("/he")` filters to `/help` — `internal/tui/autocomplete_test.go`
- [x] 3.4 RED — `TestAutocompleteNav`: up/down changes index, Enter returns selection — `internal/tui/autocomplete_test.go`
- [x] 3.5 GREEN — Add autocomplete to `InputModel`: trigger on `/`, render popup, handle selection — `internal/tui/input.go`

## Phase 1D: Clipboard Paste

- [x] 4.1 RED — `TestPasteDetection`: bracketed paste sequence inserts text — `internal/tui/input_test.go`
- [x] 4.2 GREEN — Add paste handling to `InputModel.Update()`: detect bracketed paste, insert — `internal/tui/input.go`

## Phase 1E: App Integration

- [x] 5.1 RED — `TestAppInput`: App uses `InputModel`, `Value()` returns submitted text — `internal/tui/app_test.go`
- [x] 5.2 GREEN — Replace `input string` with `InputModel` in App: struct, `NewApp`, `Update`, `View` — `internal/tui/app.go`
- [x] 5.3 RED — `TestAppKeybindings`: Tab, Enter, q, Ctrl+C still work — `internal/tui/app_test.go`
- [x] 5.4 GREEN — Add key interception layer in `App.Update()`: intercept Tab/Ctrl+C/Enter before textarea — `internal/tui/app.go`
