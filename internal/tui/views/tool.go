package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// previewLines caps collapsed tool output previews (REQ-TUI-TOOL-1).
const previewLines = 10

// ToolEvent represents a single tool invocation lifecycle: a call and
// optionally its result (REQ-TUI-TOOL-1).
type ToolEvent struct {
	CallID string
	Name   string
	Result string // empty while pending
}

// ToolModel renders the live tool-call/result list during multi-step
// turns. Each call renders as a state-bg Box (pending/success/error) with
// the call title + up to 10-line preview + expand hint; toggling expands to
// full output with inline diffs. With no events it renders "" so the layout
// budget reclaims the slot entirely (REQ-TUI-TOOL-1/2/3).
type ToolModel struct {
	events []ToolEvent
	// index by callID for fast result lookup
	byID        map[string]int
	styles      *theme.Styles
	collapse    bool // collapseToolOutput: preview + expand hint
	showDetails bool
	width       int // block width for the conversation column; 0 = unbounded
	expanded    map[string]bool
}

// SetWidth caps blocks to the conversation column.
func (m *ToolModel) SetWidth(w int) { m.width = w }

// NewToolModel creates an empty ToolModel.
func NewToolModel(styles *theme.Styles) ToolModel {
	return ToolModel{
		byID:        make(map[string]int),
		styles:      styles,
		showDetails: true,
		expanded:    make(map[string]bool),
	}
}

// AppendCall records a new tool call.
func (m *ToolModel) AppendCall(callID, name string) {
	idx := len(m.events)
	m.events = append(m.events, ToolEvent{
		CallID: callID,
		Name:   name,
	})
	m.byID[callID] = idx
}

// AppendResult attaches a result to a previously recorded call. If the
// callID is unknown, the result is silently ignored (nil-safe).
func (m *ToolModel) AppendResult(callID, result string) {
	if idx, ok := m.byID[callID]; ok && idx < len(m.events) {
		m.events[idx].Result = result
	}
}

// Toggle expands or collapses the output of a recorded call.
func (m *ToolModel) Toggle(callID string) {
	if m.expanded == nil {
		m.expanded = make(map[string]bool)
	}
	m.expanded[callID] = !m.expanded[callID]
}

// Expanded reports whether the call output is expanded.
func (m ToolModel) Expanded(callID string) bool { return m.expanded[callID] }

// SetCollapse sets collapseToolOutput mode.
func (m *ToolModel) SetCollapse(v bool) { m.collapse = v }

// SetShowDetails sets showDetails kv signal.
func (m *ToolModel) SetShowDetails(v bool) { m.showDetails = v }

// Collapse returns whether collapse is enabled.
func (m ToolModel) Collapse() bool { return m.collapse }

// ShowDetails returns showDetails state.
func (m ToolModel) ShowDetails() bool { return m.showDetails }

// CollapseOutput truncates long output with expand hint.
func CollapseOutput(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	preview := strings.Join(lines[:maxLines], "\n")
	remaining := len(lines) - maxLines
	return preview + fmt.Sprintf("\n… %d lines", remaining)
}

// boxStyle selects the state background Box for a finished or pending call.
func (m ToolModel) boxStyle(done, failed bool) lipgloss.Style {
	var style lipgloss.Style
	if m.styles != nil {
		switch {
		case !done:
			style = m.styles.ToolPendingBox
		case failed:
			style = m.styles.ToolErrorBox
		default:
			style = m.styles.ToolSuccessBox
		}
		if m.width > 0 {
			style = style.Width(m.width - 4)
		}
		return style
	}
	style = lipgloss.NewStyle().Padding(0, 1)
	if m.width > 0 {
		style = style.Width(m.width - 4)
	}
	return style
}

// Render draws one state-bg Box per tool call: title + preview(10) + expand
// hint, or the full output with inline diff when expanded (REQ-TUI-TOOL-1/3).
func (m ToolModel) Render() string {
	if len(m.events) == 0 {
		return ""
	}

	st := m.styles != nil && m.styles.Theme != nil
	var t *theme.Theme
	if st {
		t = m.styles.Theme
	}

	var parts []string
	for _, ev := range m.events {
		done := ev.Result != ""
		failed := strings.HasPrefix(ev.Result, "error:")

		title := ev.Name
		if m.showDetails && ev.CallID != "" {
			title += fmt.Sprintf(" (%s)", ev.CallID)
		}
		if !done {
			title += " · running"
		}

		if !done {
			parts = append(parts, m.boxStyle(false, false).Render(title))
			continue
		}

		body := strings.TrimPrefix(ev.Result, "error:")
		body = strings.Trim(body, "\n")
		expanded := !m.collapse || m.expanded[ev.CallID]

		var content string
		if st && strings.Contains(body, "diff --git") {
			added, removed := countDiffLines(body)
			header := fmt.Sprintf("+%d/-%d", added, removed)
			if expanded {
				content = title + " " + header + "\n" + highlightDiffResult(body, t)
			} else {
				preview := CollapseOutput(body, previewLines)
				if st {
					preview = highlightDiffResult(preview, t)
				}
				content = title + " " + header + "\n" + preview
			}
		} else if expanded {
			content = title + "\n" + body
		} else {
			content = title + "\n" + CollapseOutput(body, previewLines)
		}

		if failed && st {
			// Error state carries the error tint on the title line only;
			// the bg already signals the state.
			content = lipgloss.NewStyle().
				Foreground(lipgloss.Color(t.Error)).
				Render(title) + strings.TrimPrefix(content, title)
		}

		if m.width > 0 {
			content = truncateBlock(content, m.width-4)
		}
		parts = append(parts, m.boxStyle(true, failed).Render(content))
	}

	return strings.Join(parts, "\n")
}

// countDiffLines counts added/removed content lines for the +N/-N header.
func countDiffLines(s string) (added, removed int) {
	for _, l := range strings.Split(s, "\n") {
		switch {
		case strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++"):
			added++
		case strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---"):
			removed++
		}
	}
	return added, removed
}

// truncateBlock truncates over-wide lines so narrow terminals never panic.
func truncateBlock(s string, max int) string {
	if max <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if lipgloss.Width(l) > max {
			out := ""
			for _, r := range l {
				if lipgloss.Width(out+string(r)) > max {
					break
				}
				out += string(r)
			}
			lines[i] = out
		}
	}
	return strings.Join(lines, "\n")
}

func highlightDiffResult(s string, t *theme.Theme) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		switch {
		case strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++"):
			// Diff added with DiffAddedBg highlight
			bg := t.DiffAddedBg
			if bg == "" {
				bg = t.BGHighlight
			}
			lines[i] = lipgloss.NewStyle().Background(lipgloss.Color(bg)).Foreground(lipgloss.Color(t.DiffAdded)).Render(l)
		case strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---"):
			bg := t.DiffRemovedBg
			if bg == "" {
				bg = t.BGHighlight
			}
			lines[i] = lipgloss.NewStyle().Background(lipgloss.Color(bg)).Foreground(lipgloss.Color(t.DiffRemoved)).Render(l)
		case strings.HasPrefix(l, "@@"):
			lines[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(t.DiffHunkHeader)).Bold(true).Render(l)
		case strings.HasPrefix(l, "diff --git"):
			lines[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(t.DiffHunkHeader)).Bold(true).Render(l)
		default:
			// Context with DiffContextBg
			bg := t.DiffContextBg
			if bg != "" {
				lines[i] = lipgloss.NewStyle().Background(lipgloss.Color(bg)).Render(l)
			}
		}
	}
	return strings.Join(lines, "\n")
}
