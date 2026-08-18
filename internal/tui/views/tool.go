package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
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
	byID   map[string]int
	styles *theme.Styles
}

// NewToolModel creates an empty ToolModel.
func NewToolModel(styles *theme.Styles) ToolModel {
	return ToolModel{
		byID:   make(map[string]int),
		styles: styles,
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

// Render produces the full tool view string (REQ-TUI-TOOL-1/2).
func (m ToolModel) Render() string {
	if len(m.events) == 0 {
		return m.styles.ToolEmpty.Render("no tool calls")
	}

	var parts []string
	for _, ev := range m.events {
		var line strings.Builder
		line.WriteString(m.styles.ToolName.Render(ev.Name))

		if ev.Result != "" {
			line.WriteString(" → ")
			line.WriteString(m.styles.ToolResult.Render(ev.Result))
		} else {
			line.WriteString(" ")
			line.WriteString(m.styles.ToolPending.Render(fmt.Sprintf("(pending %s)", ev.CallID)))
		}

		parts = append(parts, line.String())
	}

	return strings.Join(parts, "\n")
}
