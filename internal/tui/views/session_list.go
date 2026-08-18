package views

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/biggs-100/kui/internal/core"
)

// sessionItem wraps a SessionMeta for display in the bubbles list.
type sessionItem struct {
	meta core.SessionMeta
}

func (i sessionItem) FilterValue() string { return i.meta.ID + " " + i.meta.Name + " " + i.meta.Summary }

// sessionItemDelegate renders a single session entry in the list.
type sessionItemDelegate struct{}

func (d sessionItemDelegate) Height() int                             { return 1 }
func (d sessionItemDelegate) Spacing() int                            { return 0 }
func (d sessionItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d sessionItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(sessionItem)
	if !ok {
		return
	}

	name := si.meta.Name
	if name == "" {
		name = si.meta.Summary
	}
	if name == "" {
		name = si.meta.ID
	}

	// Truncate name if too long
	if len(name) > 40 {
		name = name[:37] + "..."
	}

	line := fmt.Sprintf("%-40s  %-8s  %s", name, si.meta.Profile, si.meta.CreatedAt)
	if index == m.Index() {
		fmt.Fprintf(w, "▸ %s", line)
	} else {
		fmt.Fprintf(w, "  %s", line)
	}
}

// SessionListModel wraps a bubbles/list.Model for interactive session selection.
type SessionListModel struct {
	list     list.Model
	sessions []core.SessionMeta
	selected string
	quitting bool
	width    int
	height   int
}

// NewSessionListModel creates a SessionListModel from a slice of SessionMeta.
func NewSessionListModel(metas []core.SessionMeta, width, height int) SessionListModel {
	items := make([]list.Item, len(metas))
	for i, m := range metas {
		items[i] = sessionItem{meta: m}
	}

	l := list.New(items, sessionItemDelegate{}, width, height)
	l.Title = "Sessions"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)
	l.SetShowHelp(false)

	return SessionListModel{
		list:     l,
		sessions: metas,
		width:    width,
		height:   height,
	}
}

// Update handles keyboard input for the session list.
func (m SessionListModel) Update(msg tea.Msg) (SessionListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if idx := m.list.Index(); idx >= 0 && idx < len(m.sessions) {
				m.selected = m.sessions[idx].ID
			}
			return m, tea.Quit
		case tea.KeyEscape:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyRunes:
			if len(msg.Runes) == 1 && msg.Runes[0] == 'q' {
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the session list.
func (m SessionListModel) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

// Selected returns the ID of the session the user selected, or empty string
// if the user quit without selecting.
func (m SessionListModel) Selected() string {
	return m.selected
}
