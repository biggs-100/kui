package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/ui"
	"github.com/charmbracelet/lipgloss"
)

// ToolEvent represents a single tool invocation lifecycle: a call and
// optionally its result (REQ-TUI-TOOL-1).
type ToolEvent struct {
	CallID string
	Name   string
	Result string // empty while pending
}

// ToolModel renders the live tool-call/result list during multi-step
// turns. With no events it renders "" so the layout budget reclaims the
// slot (REQ-TUI-TOOL-2).
type ToolModel struct {
	events []ToolEvent
	// index by callID for fast result lookup
	byID        map[string]int
	styles      *theme.Styles
	collapse    bool // collapseToolOutput
	showDetails bool
	width       int // block width for large outputs; 0 = unbounded
}

// SetWidth caps block-style outputs to the conversation column.
func (m *ToolModel) SetWidth(w int) { m.width = w }

// NewToolModel creates an empty ToolModel.
func NewToolModel(styles *theme.Styles) ToolModel {
	return ToolModel{
		byID:        make(map[string]int),
		styles:      styles,
		showDetails: true,
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

// Render draws tool activity the upstream way: ONE INLINE ROW per call — a
// two-column state icon, the tool name (text-colored while running, muted
// once finished), and a muted one-line summary. Large/multi-line outputs
// upgrade to an indented block: invisible left bar + backgroundPanel fill,
// never a rounded box.
func (m ToolModel) Render() string {
	if len(m.events) == 0 {
		return ""
	}

	st := m.styles != nil
	t := m.styles.Theme
	if st && t == nil {
		st = false
	}

	paint := func(fg, s string) string {
		if !st || fg == "" {
			return s
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(s)
	}

	var parts []string
	for _, ev := range m.events {
		done := ev.Result != ""
		failed := strings.HasPrefix(ev.Result, "error:")

		icon, iconColor := "●", t.Warning
		nameFg := t.Text
		switch {
		case failed:
			icon, iconColor = "×", t.Error
		case done:
			icon, iconColor = "✓", t.Success
			nameFg = "" // finished tools dim through the muted path below
		}
		if !st {
			iconColor, nameFg = "", ""
		}

		name := ev.Name
		if done {
			if st {
				name = lipgloss.NewStyle().
					Foreground(lipgloss.Color(t.TextMuted)).Render(ev.Name)
			}
		} else {
			name = paint(nameFg, ev.Name)
		}
		head := paint(iconColor, icon) + " " + name
		if m.showDetails && ev.CallID != "" && st {
			head += m.styles.HomeMuted.Render(fmt.Sprintf(" (%s)", ev.CallID))
		}

		if !done {
			parts = append(parts, head+" "+paint(t.Warning, "running"))
			continue
		}

		body := strings.TrimPrefix(ev.Result, "error:")
		lineCount := strings.Count(body, "\n") + 1
		isDiff := strings.Contains(body, "diff --git")
		forceInline := m.collapse && !isDiff

		summary := strings.SplitN(strings.TrimSpace(body), "\n", 2)[0]
		if len(summary) > 60 {
			summary = summary[:57] + "..."
		}
		summaryFg := t.TextMuted
		if failed {
			summaryFg = t.Error
		}

		isBlock := !forceInline && (lineCount >= 4 || isDiff)
		if isBlock && m.width > 0 {
			hint := fmt.Sprintf("%d lines", lineCount)
			rows := []string{head + " " + paint(t.TextMuted, "· "+hint)}

			content := body
			if isDiff && st {
				content = highlightDiffResult(content, t)
			} else {
				content = CollapseOutput(content, 12)
				if failed {
					content = paint(t.Error, content)
				} else {
					content = paint(t.TextMuted, content)
				}
			}
			bar := t.Background // invisible left bar: pure indentation device
			blockStyle := lipgloss.NewStyle().
				Border(ui.SplitBorder).
				BorderForeground(lipgloss.Color(bar)).
				BorderBottom(false).
				Background(lipgloss.Color(t.BackgroundPanel)).
				Padding(1, 0, 1, 2).
				Width(m.width - 4)
			rows = append(rows, blockStyle.Render(content))
			parts = append(parts, strings.Join(rows, "\n"))
			continue
		}

		if forceInline && lineCount > 1 {
			summary = fmt.Sprintf("(%d lines)", lineCount)
			summaryFg = t.TextMuted
		}
		parts = append(parts, head+" "+paint(summaryFg, "· "+summary))
	}

	return strings.Join(parts, "\n")
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
		default:
			// Context with DiffContextBg
			bg := t.DiffContextBg
			if bg != "" {
				lines[i] = lipgloss.NewStyle().Background(lipgloss.Color(bg)).Render(l)
			}
		}
	}
	// If collapsed, truncate after highlight
	return strings.Join(lines, "\n")
}
