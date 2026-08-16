package views

import (
	"fmt"
	"strings"

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
	byID map[string]int
}

// NewToolModel creates an empty ToolModel.
func NewToolModel() ToolModel {
	return ToolModel{
		byID: make(map[string]int),
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

var (
	toolNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	toolResultStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	toolPendingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Faint(true)

	toolEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Faint(true)
)

// Render produces the full tool view string (REQ-TUI-TOOL-1/2).
func (m ToolModel) Render() string {
	if len(m.events) == 0 {
		return toolEmptyStyle.Render("no tool calls")
	}

	var parts []string
	for _, ev := range m.events {
		var line strings.Builder
		line.WriteString(toolNameStyle.Render(ev.Name))

		if ev.Result != "" {
			line.WriteString(" → ")
			line.WriteString(toolResultStyle.Render(ev.Result))
		} else {
			line.WriteString(" ")
			line.WriteString(toolPendingStyle.Render(fmt.Sprintf("(pending %s)", ev.CallID)))
		}

		parts = append(parts, line.String())
	}

	return strings.Join(parts, "\n")
}
