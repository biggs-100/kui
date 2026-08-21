package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
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
// turns. When no events exist (nil observer or no tool calls), it shows
// an empty-state hint (REQ-TUI-TOOL-2).
type ToolModel struct {
	events []ToolEvent
	// index by callID for fast result lookup
	byID        map[string]int
	styles      *theme.Styles
	collapse    bool // collapseToolOutput
	showDetails bool
}

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

// Render produces the full tool view string (REQ-TUI-TOOL-1/2).
// Supports collapseToolOutput and showDetails toggles, and diff highlight
// backgrounds via theme Diff*Bg tokens.
func (m ToolModel) Render() string {
	if len(m.events) == 0 {
		if m.styles != nil {
			return m.styles.HomeMuted.Render("no tool calls")
		}
		return "no tool calls"
	}

	var parts []string
	for _, ev := range m.events {
		var line strings.Builder
		// Per-tool metadata: show Name and CallID when showDetails true
		nameStr := ev.Name
		if m.styles != nil {
			nameStr = m.styles.ToolName.Render(ev.Name)
		}
		line.WriteString(nameStr)
		if m.showDetails && ev.CallID != "" {
			meta := fmt.Sprintf(" (%s)", ev.CallID)
			if m.styles != nil {
				meta = m.styles.HomeMuted.Render(meta)
			}
			line.WriteString(meta)
		}

		if ev.Result != "" {
			if !m.showDetails {
				// detail rows hidden when showDetails=false
				line.WriteString(" ")
				hidden := "— details hidden"
				if m.styles != nil {
					hidden = m.styles.HomeMuted.Render(hidden)
				}
				line.WriteString(hidden)
			} else {
				result := ev.Result
				// Diff highlight detection: if result looks like diff, use Diff*Bg
				isDiff := strings.Contains(result, "diff --git") || strings.Contains(ev.Name, "diff") || (strings.Contains(result, "\n+") || strings.Contains(result, "\n-"))
				if isDiff && m.styles != nil && m.styles.Theme != nil {
					// Apply diff highlight backgrounds per-line token colors
					result = highlightDiffResult(result, m.styles.Theme)
				} else {
					// Normal result handling
					if m.collapse {
						result = CollapseOutput(result, 10)
					} else {
						if len(result) > 200 {
							result = result[:200] + "…"
						}
						result = strings.ReplaceAll(result, "\n", " ")
					}
				}
				// When collapsed diff, still need hint; CollapseOutput already adds hint
				// Ensure collapsed output truncates correctly for long outputs (500 lines case)
				if m.collapse && !isDiff {
					// Already handled via CollapseOutput above
				}
				// For non-diff collapsed, CollapseOutput replaced newlines; for expanded, replace newlines with spaces
				if !m.collapse && !isDiff {
					result = strings.ReplaceAll(result, "\n", " ")
				}
				line.WriteString(" → ")
				if m.styles != nil {
					line.WriteString(m.styles.ToolResult.Render(result))
				} else {
					line.WriteString(result)
				}
				// When collapsed and diff, ensure hint present if truncated
				if m.collapse && isDiff && strings.Count(ev.Result, "\n") > 10 && !strings.Contains(result, "…") {
					line.WriteString(fmt.Sprintf(" … %d lines", strings.Count(ev.Result, "\n")-10))
				}
			}
		} else {
			line.WriteString(" ")
			pending := "○ pending"
			if m.styles != nil {
				pending = m.styles.ToolPending.Render(pending)
			}
			line.WriteString(pending)
		}

		inner := line.String()
		// Wrap each entry in bordered panel using backgroundPanel token
		if m.styles != nil {
			panel := m.styles.Panel.Render(inner)
			parts = append(parts, panel)
		} else {
			parts = append(parts, inner)
		}
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
