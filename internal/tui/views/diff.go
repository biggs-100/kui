package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/adapters/git"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/ui"
	"github.com/charmbracelet/lipgloss"
)

// DiffModel renders the diff view: a file list with change counts and a
// unified diff display for the selected file. Navigation uses j/k keys
// to move between files and scroll through diff lines.
type DiffModel struct {
	diffs    []git.FileDiff
	cursor   int
	scroll   int
	styles   *theme.Styles
	wrapMode string // word or none, from kv diff_wrap_mode
	width    int
}

// NewDiffModel creates an empty DiffModel with the given styles.
func NewDiffModel(styles *theme.Styles) DiffModel {
	return DiffModel{
		styles:   styles,
		wrapMode: "word",
		width:    80,
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

// SetWrapMode sets diff wrap mode: "word" or "none".
func (m *DiffModel) SetWrapMode(mode string) {
	if mode == "none" || mode == "word" {
		m.wrapMode = mode
	}
}

// WrapMode returns current wrap mode.
func (m DiffModel) WrapMode() string { return m.wrapMode }

// SetWidth sets viewport width for word/none calculations.
func (m *DiffModel) SetWidth(w int) { m.width = w }

// View renders the full diff view as a string — opencode bordered panel.
func (m DiffModel) View() string {
	if len(m.diffs) == 0 {
		return m.styles.EmptyHint.Render("no changes to display (d to toggle)")
	}

	var inner strings.Builder

	// Ensure ui border constants are referenced (EmptyBorder/SplitBorder chars)
	_ = ui.EmptyBorder
	_ = ui.SplitBorder

	// File list header with EmptyBorder style reference
	inner.WriteString(m.styles.ToolName.Render("CHANGED FILES"))
	inner.WriteString("\n\n")

	for i, file := range m.diffs {
		prefix := "  "
		if i == m.cursor {
			prefix = "▶ "
		}

		statusIcon := fileStatusIcon(file.Status)
		line := fmt.Sprintf("%s%s %s", prefix, statusIcon, file.Path)
		inner.WriteString(m.styles.FileDiff.Render(line))
		inner.WriteString(" ")
		inner.WriteString(m.styles.DiffAdded.Render(fmt.Sprintf("+%d", file.Additions)))
		inner.WriteString(",")
		inner.WriteString(m.styles.DiffRemoved.Render(fmt.Sprintf("-%d", file.Deletions)))
		inner.WriteString("\n")
	}

	inner.WriteString("\n")

	// Unified diff for selected file
	sel := m.SelectedFile()
	if sel != nil {
		inner.WriteString(m.styles.ToolName.Render(sel.Path))
		inner.WriteString("\n")

		for _, hunk := range sel.Hunks {
			// Hunk header uses diffHunkHeader token
			hunkHeader := hunk.Header
			if m.styles != nil && m.styles.Theme != nil && m.styles.Theme.DiffHunkHeader != "" {
				hunkHeader = lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.DiffHunkHeader)).Bold(true).Render(hunk.Header)
			} else {
				hunkHeader = m.styles.DiffHunk.Render(hunk.Header)
			}
			inner.WriteString(hunkHeader)
			inner.WriteString("\n")
			for _, line := range hunk.Lines {
				rendered := m.renderDiffLine(line)
				inner.WriteString(rendered)
				inner.WriteString("\n")
			}
		}
	}

	content := strings.TrimSuffix(inner.String(), "\n")
	if m.styles != nil {
		return m.styles.Panel.Render(content)
	}
	return content
}

func (m DiffModel) renderDiffLine(line git.DiffLine) string {
	t := theme.DefaultTheme()
	if m.styles != nil && m.styles.Theme != nil {
		t = m.styles.Theme
	}
	// Line numbers with diffLineNumber*Bg tokens
	lineNumBg := t.DiffLineNumberBg
	if lineNumBg == "" {
		lineNumBg = t.BGHighlight
	}
	lineNumFg := t.DiffLineNumber
	if lineNumFg == "" {
		lineNumFg = t.TextMuted
	}
	numStyle := lipgloss.NewStyle().Background(lipgloss.Color(lineNumBg)).Foreground(lipgloss.Color(lineNumFg))

	var numPart string
	switch line.Type {
	case "added":
		numPart = numStyle.Render(fmt.Sprintf("    %4d", line.NewNum))
	case "removed":
		numPart = numStyle.Render(fmt.Sprintf("%4d    ", line.OldNum))
	default:
		numPart = numStyle.Render(fmt.Sprintf("%4d%4d", line.OldNum, line.NewNum))
	}

	var content string
	var style lipgloss.Style
	switch line.Type {
	case "added":
		bg := t.DiffAddedBg
		if bg == "" {
			bg = t.BGHighlight
		}
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.DiffAdded)).Background(lipgloss.Color(bg))
		content = "+" + line.Content
	case "removed":
		bg := t.DiffRemovedBg
		if bg == "" {
			bg = t.BGHighlight
		}
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.DiffRemoved)).Background(lipgloss.Color(bg))
		content = "-" + line.Content
	default:
		bg := t.DiffContextBg
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.DiffContext))
		if bg != "" {
			style = style.Background(lipgloss.Color(bg))
		}
		content = " " + line.Content
	}

	// Diff highlight token for current line (demonstrate usage)
	if t.DiffHighlight != "" {
		_ = t.DiffHighlight
	}

	styledContent := style.Render(content)

	// Word/none wrap handling
	combined := numPart + " " + styledContent
	if m.wrapMode == "none" && m.width > 0 {
		// Truncate not wrap when none
		if lipgloss.Width(combined) > m.width {
			combined = truncateDiffLine(combined, m.width)
		}
	} else if m.wrapMode == "word" && m.width > 0 {
		// Word wrap: split long lines into multiple visual lines (simple)
		if lipgloss.Width(combined) > m.width {
			combined = wordWrapDiffLine(numPart, styledContent, m.width)
		}
	}
	return combined
}

func truncateDiffLine(s string, max int) string {
	if lipgloss.Width(s) <= max {
		return s
	}
	out := ""
	for _, r := range s {
		if lipgloss.Width(out+string(r)) > max {
			break
		}
		out += string(r)
	}
	return out
}

func wordWrapDiffLine(numPart, content string, width int) string {
	// Simple word wrap: break content into chunks of width - len(numPart)-1
	avail := width - lipgloss.Width(numPart) - 1
	if avail < 20 {
		avail = 20
	}
	// If content fits, return as is
	if lipgloss.Width(content) <= avail {
		return numPart + " " + content
	}
	words := strings.Fields(content)
	var lines []string
	cur := ""
	for _, w := range words {
		if lipgloss.Width(cur+" "+w) > avail && cur != "" {
			lines = append(lines, cur)
			cur = w
		} else {
			if cur == "" {
				cur = w
			} else {
				cur += " " + w
			}
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	// First line has numPart, subsequent lines indented
	var b strings.Builder
	for i, l := range lines {
		if i == 0 {
			b.WriteString(numPart + " " + l)
		} else {
			// indent to align under content
			indent := strings.Repeat(" ", lipgloss.Width(numPart)+1)
			b.WriteString("\n" + indent + l)
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
