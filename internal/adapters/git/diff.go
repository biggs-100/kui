// Package git implements the GitAdapter port for kui's diff rendering.
// It parses `git diff` output into structured FileDiff/Hunk/DiffLine types.
package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

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

// DiffCommand runs `git diff` in the given directory and parses the output.
// It works against the working tree (unstaged changes).
func DiffCommand(dir string) ([]FileDiff, error) {
	cmd := exec.Command("git", "diff", "--no-color", "-p")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return parseDiff(string(out))
}

// parseDiff parses unified diff output into FileDiff slices.
func parseDiff(output string) ([]FileDiff, error) {
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}

	var files []FileDiff
	var current *FileDiff
	var currentHunk *Hunk
	oldNum, newNum := 0, 0

	lines := strings.Split(output, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// New file header
		if strings.HasPrefix(line, "diff --git ") {
			if current != nil {
				files = append(files, *current)
			}
			// Extract path from "diff --git a/path b/path"
			path := parseGitDiffPath(line)
			current = &FileDiff{
				Path:   path,
				Status: "modified",
			}
			currentHunk = nil
			continue
		}

		if current == nil {
			continue
		}

		// File status markers
		if strings.HasPrefix(line, "new file mode ") {
			current.Status = "added"
			continue
		}
		if strings.HasPrefix(line, "deleted file mode ") {
			current.Status = "deleted"
			continue
		}
		if strings.HasPrefix(line, "rename from ") {
			current.Status = "renamed"
			continue
		}

		// Hunk header: @@ -oldStart,oldCount +newStart,newCount @@
		if strings.HasPrefix(line, "@@ ") {
			if currentHunk != nil {
				current.Hunks = append(current.Hunks, *currentHunk)
			}
			oldStart, newStart := parseHunkHeader(line)
			currentHunk = &Hunk{
				Header:   line,
				OldStart: oldStart,
				NewStart: newStart,
			}
			oldNum = oldStart
			newNum = newStart
			continue
		}

		// Diff lines (must be inside a hunk)
		if currentHunk != nil {
			if len(line) == 0 {
				continue
			}
			switch line[0] {
			case '+':
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    "added",
					Content: line[1:],
					OldNum:  0,
					NewNum:  newNum,
				})
				current.Additions++
				newNum++
			case '-':
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    "removed",
					Content: line[1:],
					OldNum:  oldNum,
					NewNum:  0,
				})
				current.Deletions++
				oldNum++
			default:
				// Context line (starts with space or is empty inside hunk)
				content := line
				if len(line) > 0 && line[0] == ' ' {
					content = line[1:]
				}
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    "context",
					Content: content,
					OldNum:  oldNum,
					NewNum:  newNum,
				})
				oldNum++
				newNum++
			}
		}
	}

	// Flush last file/hunk
	if currentHunk != nil {
		current.Hunks = append(current.Hunks, *currentHunk)
	}
	if current != nil {
		files = append(files, *current)
	}

	return files, nil
}

// parseGitDiffPath extracts the file path from a "diff --git a/path b/path" line.
func parseGitDiffPath(line string) string {
	// Format: diff --git a/path b/path
	// We take the b/ path as it's the post-image.
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return ""
	}
	bPath := parts[3]
	if strings.HasPrefix(bPath, "b/") {
		return bPath[2:]
	}
	return bPath
}

// parseHunkHeader extracts old and new start line numbers from a hunk header.
// Format: @@ -oldStart,oldCount +newStart,newCount @@ optional context
func parseHunkHeader(line string) (oldStart, newStart int) {
	// Find the two @@ markers
	after := strings.TrimPrefix(line, "@@ ")
	// Split on " @@" to isolate the numbers
	idx := strings.Index(after, " @@")
	if idx < 0 {
		return 0, 0
	}
	numbers := after[:idx]
	// Split on " +" to separate old and new
	parts := strings.SplitN(numbers, " +", 2)
	if len(parts) != 2 {
		return 0, 0
	}

	oldStart = parseLineNum(strings.TrimPrefix(parts[0], "-"))
	newStart = parseLineNum(strings.TrimPrefix(parts[1], "+"))
	return oldStart, newStart
}

// parseLineNum extracts the starting line number from "start,count" or "start".
func parseLineNum(s string) int {
	parts := strings.SplitN(s, ",", 2)
	if len(parts) == 0 {
		return 0
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return n
}
