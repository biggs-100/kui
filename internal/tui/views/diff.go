package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/adapters/git"
	"github.com/biggs-100/kui/internal/tui/theme"
)

// DiffModel renders the diff view: a file list with change counts and a
// unified diff display for the selected file. Navigation uses j/k keys
// to move between files and scroll through diff lines.
type DiffModel struct {
	diffs   []git.FileDiff
	cursor  int
	scroll  int
	styles  *theme.Styles
}

// NewDiffModel creates an empty DiffModel with the given styles.
func NewDiffModel(styles *theme.Styles) DiffModel {
	return DiffModel{
		styles: styles,
	}
}

// SetDiffs replaces the file list with the given diffs and resets the cursor.
func (m *DiffModel) SetDiffs(diffs []git.FileDiff) {
	m.diffs = diffs
	m.cursor = 0
	m.scroll = 0
}

// Files returns the current file diffs (for inspection).
func (m DiffModel) Files() []git.FileDiff {
	return m.diffs
}

// Cursor returns the current cursor position in the file list.
func (m DiffModel) Cursor() int {
	return m.cursor
}

// SelectedFile returns the currently selected file diff, or nil if empty.
func (m DiffModel) SelectedFile() *git.FileDiff {
	if len(m.diffs) == 0 {
		return nil
	}
	return &m.diffs[m.cursor]
}

// MoveDown moves the cursor down one position, clamping at the end.
func (m *DiffModel) MoveDown() {
	if m.cursor < len(m.diffs)-1 {
		m.cursor++
		m.scroll = 0
	}
}

// MoveUp moves the cursor up one position, clamping at 0.
func (m *DiffModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
		m.scroll = 0
	}
}

// View renders the full diff view as a string.
func (m DiffModel) View() string {
	if len(m.diffs) == 0 {
		return m.styles.EmptyHint.Render("no changes to display (d to toggle)")
	}

	var b strings.Builder

	// File list header
	b.WriteString(m.styles.ToolName.Render("CHANGED FILES"))
	b.WriteString("\n\n")

	for i, file := range m.diffs {
		prefix := "  "
		if i == m.cursor {
			prefix = "▶ "
		}

		statusIcon := fileStatusIcon(file.Status)
		addDel := fmt.Sprintf("+%d,-%d", file.Additions, file.Deletions)

		line := fmt.Sprintf("%s%s %s", prefix, statusIcon, file.Path)
		b.WriteString(m.styles.FileDiff.Render(line))
		b.WriteString(" ")
		b.WriteString(m.styles.DiffAdded.Render(fmt.Sprintf("+%d", file.Additions)))
		b.WriteString(",")
		b.WriteString(m.styles.DiffRemoved.Render(fmt.Sprintf("-%d", file.Deletions)))
		b.WriteString(" ")
		b.WriteString(m.styles.DiffContext.Render(addDel))
		b.WriteString("\n")
	}

	// Separator
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	// Unified diff for selected file
	sel := m.SelectedFile()
	if sel == nil {
		return b.String()
	}

	b.WriteString(m.styles.ToolName.Render(sel.Path))
	b.WriteString("\n")

	for _, hunk := range sel.Hunks {
		b.WriteString(m.styles.DiffHunk.Render(hunk.Header))
		b.WriteString("\n")
		for _, line := range hunk.Lines {
			switch line.Type {
			case "added":
				b.WriteString(m.styles.DiffAdded.Render("+" + line.Content))
			case "removed":
				b.WriteString(m.styles.DiffRemoved.Render("-" + line.Content))
			default:
				b.WriteString(m.styles.DiffContext.Render(" " + line.Content))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// fileStatusIcon returns a short icon for the file status.
func fileStatusIcon(status string) string {
	switch status {
	case "added":
		return "A"
	case "deleted":
		return "D"
	case "renamed":
		return "R"
	default:
		return "M"
	}
}
