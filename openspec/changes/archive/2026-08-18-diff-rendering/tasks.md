# Tasks: Diff Rendering

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~345 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Git adapter: types + diff parsing | PR 1 | `go test ./internal/adapters/git/ -run TestGitDiff` | N/A — unit tests only, no runtime scenario needed | `internal/core/git.go`, `internal/adapters/git/diff.go`, `internal/adapters/git/diff_test.go` |
| 2 | Diff view + app integration | PR 1 | `go test ./internal/tui/... -run TestDiff` | N/A — Bubbletea model tests, no interactive flow | `internal/tui/views/diff.go`, `internal/tui/views/diff_test.go`, `internal/tui/app.go`, `internal/tui/app_test.go`, `internal/tui/theme/styles.go` |

## Phase 1: Git Adapter — Types and Diff Parsing

- [x] 1.1 RED — Write `TestGitDiffParse` in `internal/adapters/git/diff_test.go`: table-driven test that parses a sample unified diff string into `[]FileDiff` with correct Path, Status, Additions, Deletions, and Hunks
- [x] 1.2 GREEN — Create `internal/core/git.go` with `GitAdapter` interface, `FileDiff`, `Hunk`, `DiffLine` types (port, no implementation dependency)
- [x] 1.3 GREEN — Create `internal/adapters/git/diff.go` with `parseDiff(output string) ([]FileDiff, error)` function implementing unified diff parsing (header, hunk, line types)
- [x] 1.4 RED — Write `TestGitDiffCommand` in `internal/adapters/git/diff_test.go`: integration test calling real `git diff` in a temp repo; skip with `testing.Short()`
- [x] 1.5 GREEN — Add `DiffCommand(dir string) ([]FileDiff, error)` to `internal/adapters/git/diff.go`: runs `git diff --no-color -p`, pipes through `parseDiff`

## Phase 2: Diff View — DiffModel and Rendering

- [x] 2.1 RED — Write `TestDiffModelCreate` in `internal/tui/views/diff_test.go`: creates empty `DiffModel` via `NewDiffModel(styles)`, asserts empty file list and zero cursor
- [x] 2.2 GREEN — Create `internal/tui/views/diff.go` with `DiffModel` struct: file list, selected index, scroll offset, `NewDiffModel`, `SetDiffs`, `Update`, `View` methods
- [x] 2.3 RED — Write `TestDiffRender` in `internal/tui/views/diff_test.go`: feeds sample `[]FileDiff` to model, asserts `View()` output contains `+`/`-` prefixes and `@@` hunk headers

## Phase 3: App Integration — Toggle and Theme

- [x] 3.1 GREEN — Add diff styles to `internal/tui/theme/styles.go`: `DiffAdded` (green), `DiffRemoved` (red), `DiffContext` (muted), `DiffHunk` (accent), `DiffFile` (primary) using existing `Theme.DiffAdded`/`DiffRemoved`/`DiffContext` colors
- [x] 3.2 RED — Write `TestAppDiffToggle` in `internal/tui/app_test.go`: sends `tea.KeyMsg{Type: 'd'}` to App, asserts diff view becomes visible; sends again, asserts hidden
- [x] 3.3 GREEN — Modify `internal/tui/app.go`: add `diff views.DiffModel` and `diffVisible bool` fields, add 'd' key handler in Update, render diff panel when visible
