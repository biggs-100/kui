# Proposal: diff-rendering

## Intent

Add a diff rendering system to kui's TUI that shows file changes with syntax-highlighted unified diffs, a file tree with change counts, hunk navigation, and revert support. This enables users to see exactly what code changed during agent operations without leaving the TUI.

## Current Gap

kui has no diff display capability. When the agent modifies files via tools (write_file, bash), users cannot see what changed. The TUI shows tool calls and results as plain text, losing the structured information that diffs provide. OpenCode has a full diff viewer with split/unified toggle, file tree, hunk navigation, and revert support — kui has none of this.

The theme system already defines diff colors (`DiffAdded`, `DiffRemoved`, `DiffContext`) but they are unused. The TUI architecture supports adding new views (chat, tool, header, footer are separate models).

## Proposed Solution

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    TUI App                               │
├─────────────────────────────────────────────────────────┤
│  Header (profile tabs)                                  │
├─────────────────────────────────────────────────────────┤
│  Chat View │ Diff View (new)                            │
│            │ ┌─────────────────────────────────────────┐│
│            │ │ File Tree (left panel)                  ││
│            │ │  file1.go (+12, -3)                     ││
│            │ │  file2.go (+5, -8)                      ││
│            │ ├─────────────────────────────────────────┤│
│            │ │ Unified Diff (right panel)              ││
│            │ │ @@ -10,7 +10,8 @@                      ││
│            │ │  context line                           ││
│            │ │ +added line                             ││
│            │ │ -removed line                           ││
│            │ └─────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────┤
│  Tool View                                              │
├─────────────────────────────────────────────────────────┤
│  Input Line                                             │
└─────────────────────────────────────────────────────────┘
```

### Components

1. **Git Adapter** (`internal/adapters/git/`)
   - `diff.go`: Parse `git diff` output into structured `FileDiff` and `Hunk` types
   - `status.go`: Get file change status (modified, added, deleted, renamed)
   - `revert.go`: Revert file changes via `git checkout`
   - Port interface in `internal/core/git.go` for testability

2. **Diff View** (`internal/tui/views/diff.go`)
   - `DiffModel`: Renders file tree and unified diff
   - File tree with `+N, -M` change counts
   - Unified diff with syntax-highlighted additions/removals
   - Hunk navigation (j/k keys)
   - File selection (enter key)
   - Revert support (r key)

3. **Theme Integration** (`internal/tui/theme/styles.go`)
   - Add diff-specific styles using existing theme colors:
     - `DiffAdded`: green text for `+` lines
     - `DiffRemoved`: red text for `-` lines
     - `DiffContext`: muted text for context lines
     - `DiffHunk`: accent color for `@@` headers
     - `DiffFile`: primary color for file names

4. **App Integration** (`internal/tui/app.go`)
   - Add diff view as toggleable panel (Tab key cycles through views)
   - Listen for file change events from tools
   - Auto-show diff view after file modifications

### Data Flow

```
Agent Tool Execution
    ↓
Git Adapter (parse diff)
    ↓
Diff Model (structured data)
    ↓
Diff View (render)
    ↓
TUI App (display)
```

### Key Types

```go
// internal/core/git.go
type FileDiff struct {
    Path    string
    Status  FileStatus // added, modified, deleted, renamed
    Additions int
    Deletions int
    Hunks   []Hunk
}

type Hunk struct {
    Header    string
    Lines     []DiffLine
    OldStart  int
    NewStart  int
}

type DiffLine struct {
    Type    DiffLineType // added, removed, context
    Content string
    OldNum  int
    NewNum  int
}

type GitAdapter interface {
    Diff() ([]FileDiff, error)
    Status() ([]FileStatus, error)
    Revert(path string) error
}
```

## Scope

### In Scope
- Unified diff view with syntax highlighting
- File tree with change counts (+/-)
- Hunk navigation (j/k keys)
- File selection (enter key)
- Revert support (r key)
- Git adapter for diff parsing
- Theme integration for diff colors

### Out of Scope (Future)
- Split diff view (side-by-side)
- Commit support
- Branch management
- Staging support
- Interactive rebase

## Success Criteria

1. **Functional**: Users can view file changes after agent operations
2. **Usable**: Diff view is accessible via keyboard shortcuts
3. **Performant**: Diff rendering completes in <100ms for files up to 1000 lines
4. **Integrated**: Diff view follows existing TUI patterns and theme system
5. **Testable**: Git adapter has unit tests, diff view has render tests

## Risks

1. **Git Dependency**: Requires git to be installed and repository to be initialized
   - Mitigation: Graceful degradation when git unavailable, show "no changes" message

2. **Performance**: Large diffs may cause rendering lag
   - Mitigation: Virtual scrolling, limit visible hunks, lazy loading

3. **Complexity**: Diff parsing is non-trivial (rename detection, binary files)
   - Mitigation: Start with simple unified diff, iterate on edge cases

4. **TUI Layout**: Adding diff view may disrupt existing layout
   - Mitigation: Make diff view toggleable, preserve existing three-region layout as default

## Rollback Plan

1. Remove diff view from app.go
2. Remove git adapter from adapters/
3. Remove diff styles from theme
4. Remove core/git.go types
5. All changes are additive — no existing functionality modified

## Implementation Phases

### Phase 1: Git Adapter (Core)
- Add `internal/core/git.go` with types
- Add `internal/adapters/git/diff.go` with parser
- Add `internal/adapters/git/status.go`
- Unit tests for diff parsing

### Phase 2: Diff View (TUI)
- Add `internal/tui/views/diff.go` with DiffModel
- Add diff styles to `internal/tui/theme/styles.go`
- Integrate diff view into app.go
- Render tests

### Phase 3: Navigation & Interaction
- Add hunk navigation (j/k)
- Add file selection (enter)
- Add revert support (r)
- Keyboard shortcut tests

### Phase 4: Polish
- Auto-show diff after file changes
- Performance optimization for large diffs
- Edge case handling (binary files, renames)
