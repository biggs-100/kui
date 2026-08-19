// Package core provides domain types for kui. This file defines the Git
// port interface and diff-related types. The port lives in core to keep
// the hexagonal dependency rule: adapters depend on core, never the reverse.
package core

// GitAdapter is the port for interacting with a Git repository. Concrete
// implementations live in adapters/git/. The TUI depends on this interface,
// not the implementation.
type GitAdapter interface {
	Diff() ([]FileDiff, error)
	Revert(path string) error
}

// FileDiff represents the changes in a single file.
type FileDiff struct {
	Path      string
	Status    string // "modified", "added", "deleted", "renamed"
	Additions int
	Deletions int
	Hunks     []Hunk
}

// Hunk represents a contiguous block of changes within a file.
type Hunk struct {
	Header   string
	OldStart int
	NewStart int
	Lines    []DiffLine
}

// DiffLine is a single line within a hunk, tagged by its type.
type DiffLine struct {
	Type    string // "added", "removed", "context"
	Content string
	OldNum  int
	NewNum  int
}
