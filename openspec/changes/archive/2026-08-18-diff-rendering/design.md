# Design: Diff Rendering

## Technical Approach

Add a unified diff viewer to kui's TUI using the existing hexagonal architecture pattern. A new Git adapter in `internal/adapters/git/` parses `git diff` output into structured types defined in `internal/core/`. A new `DiffModel` view in `internal/tui/views/` renders the diff, integrated as a toggleable panel in the App alongside the existing chat view.

## Architecture Decisions

### Decision: Diff Parsing Strategy

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Pure Go parser (no git binary) | Zero dependency, but must reimplement diff format | Rejected — high complexity, low payoff |
| Shell out to `git diff` | Simple implementation, requires git installed | **Chosen** — git is a hard dependency for repos anyway |
| Hybrid: try git, fallback to manual parsing | Most robust, highest complexity | Rejected — premature optimization for Phase 1 |

### Decision: Diff View Layout

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Horizontal split (file tree left, diff right) | Familiar UX, requires width-aware layout | Rejected — tight on narrow terminals |
| Vertical split (file tree top, diff bottom) | Simple layout, scrollable sections | Rejected — reduces diff visibility |
| Tabbed file tree / diff panels | Single focus, minimal layout complexity | **Chosen** — matches existing view pattern |

### Decision: View Integration

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Always-visible panel | Persistent visibility, reduces chat space | Rejected — chat is primary interaction |
| Toggleable panel via keybind | On-demand, preserves layout | **Chosen** — non-intrusive, matches session list pattern |
| Replace chat view entirely | Maximum diff space, loses context | Rejected — users need both |

## Data Flow

```
Agent Tool Execution (write_file, bash)
    ↓
Controller emits toolResultMsg
    ↓
App detects file-change tool → calls gitAdapter.Diff()
    ↓
DiffModel receives []FileDiff
    ↓
User presses 'd' → toggles diff view
    ↓
DiffModel.Render() → styled diff output
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/core/git.go` | Create | Git port interface + FileDiff/Hunk/DiffLine types |
| `internal/adapters/git/diff.go` | Create | Git adapter: parse `git diff --no-color --stat -p` output |
| `internal/adapters/git/diff_test.go` | Create | Unit tests with golden files for diff parsing |
| `internal/tui/views/diff.go` | Create | DiffModel: file list, unified diff rendering, keyboard nav |
| `internal/tui/views/diff_test.go` | Create | Render tests and keyboard handling |
| `internal/tui/theme/styles.go` | Modify | Add diff-specific styles: DiffAdded, DiffRemoved, DiffHunk |
| `internal/tui/app.go` | Modify | Add diff view field, toggle logic, event routing |
| `internal/tui/app_test.go` | Modify | Test diff toggle keybinding |

## Interfaces / Contracts

```go
// internal/core/git.go — port (no implementation dependency)
type GitAdapter interface {
    Diff() ([]FileDiff, error)
    Revert(path string) error
}

type FileDiff struct {
    Path      string
    Status    string   // "modified", "added", "deleted", "renamed"
    Additions int
    Deletions int
    Hunks     []Hunk
}

type Hunk struct {
    Header   string
    OldStart int
    NewStart int
    Lines    []DiffLine
}

type DiffLine struct {
    Type    string // "added", "removed", "context"
    Content string
    OldNum  int
    NewNum  int
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Diff parser output for standard unified diff | Table-driven tests with golden files |
| Unit | DiffModel render output | Assert styled string contains expected elements |
| Unit | App toggle keybind | Verify view state changes on 'd' key |
| Integration | Git adapter → DiffModel flow | Mock git adapter, verify end-to-end render |
| E2E | Manual: run agent, press 'd', verify diff visible | Smoke test — automated when test harness supports TUI |

## Threat Matrix

N/A — no routing, shell commands, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. The git adapter calls `git diff` and `git checkout` as a standard subprocess with no user input injection risk.

## Migration / Rollout

No migration required. This is purely additive — new files and view toggle. Existing layout and behavior unchanged when diff view is inactive.

## Open Questions

- [ ] Should the diff view auto-open after tool execution, or require manual toggle? (Current design: manual toggle via 'd' key)
- [ ] Should revert use `git checkout HEAD -- <file>` (discard all changes) or `git checkout <file>` (discard unstaged)?
- [ ] Should the file tree show all files or only changed files? (Proposed: only changed files)
