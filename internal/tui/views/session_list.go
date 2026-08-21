package views

import (
	"github.com/biggs-100/kui/internal/core"
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/ui"
	"github.com/biggs-100/kui/internal/tui/util"
	tea "github.com/charmbracelet/bubbletea"
)

// SessionListModel wraps DialogSelect for interactive session selection with
// 76 truncate, Esc filter→close, preserveSelection, scrollAcceleration.
type SessionListModel struct {
	ds       *ui.DialogSelect[string]
	sessions []core.SessionMeta
	selected string
	quitting bool
	width    int
	height   int
	styles   *theme.Styles
}

func sessionTitle(m core.SessionMeta) string {
	name := m.Name
	if name == "" {
		name = m.Summary
	}
	if name == "" {
		name = m.ID
	}
	return util.TruncateMiddle(name, 76)
}

func buildSessionItems(metas []core.SessionMeta) []ui.SelectItem[string] {
	items := make([]ui.SelectItem[string], 0, len(metas))
	for _, m := range metas {
		title := sessionTitle(m)
		cat := m.Profile
		if cat == "" {
			cat = "Other"
		}
		detail := m.CreatedAt
		// also show truncated ID as detail if createdAt empty
		if detail == "" {
			detail = util.TruncateMiddle(m.ID, 76)
		}
		detail = util.TruncateMiddle(detail, 76)
		items = append(items, ui.SelectItem[string]{
			Title:    title,
			Category: cat,
			Detail:   detail,
			Value:    m.ID,
		})
	}
	return items
}

// NewSessionListModel creates a SessionListModel from a slice of SessionMeta.
func NewSessionListModel(metas []core.SessionMeta, width, height int) SessionListModel {
	items := buildSessionItems(metas)
	ds := ui.NewDialogSelect(items, width, height)
	ds.SetFlat(false)
	ds.SetEmptyView("No sessions")
	return SessionListModel{
		ds:       ds,
		sessions: metas,
		width:    width,
		height:   height,
		styles:   theme.NewStyles(theme.OpenCode()),
	}
}

// SetStyles sets theme styles.
func (m *SessionListModel) SetStyles(s *theme.Styles) {
	if s != nil {
		m.styles = s
		m.ds.SetStyles(s)
	}
}

// Update handles keyboard input for the session list.
func (m SessionListModel) Update(msg tea.Msg) (SessionListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if it := m.ds.SelectedItem(); it != nil {
				m.selected = it.Value
			}
			return m, tea.Quit
		case tea.KeyEscape:
			if !m.ds.HandleEsc() {
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case tea.KeyBackspace:
			ft := m.ds.FilterText()
			if len(ft) > 0 {
				m.ds.Filter(ft[:len(ft)-1])
			}
			return m, nil
		case tea.KeyRunes:
			if len(msg.Runes) == 1 && msg.Runes[0] == 'q' && m.ds.FilterText() == "" {
				m.quitting = true
				return m, tea.Quit
			}
			ft := m.ds.FilterText() + string(msg.Runes)
			m.ds.Filter(ft)
			return m, nil
		case tea.KeyUp:
			m.ds.MoveUp()
			return m, nil
		case tea.KeyDown:
			m.ds.MoveDown()
			return m, nil
		}
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'k':
				if m.ds.FilterText() == "" {
					m.ds.MoveUp()
					return m, nil
				}
			case 'j':
				if m.ds.FilterText() == "" {
					m.ds.MoveDown()
					return m, nil
				}
			}
		}
	}
	return m, nil
}

// View renders the session list.
func (m SessionListModel) View() string {
	if m.quitting {
		return ""
	}
	m.ds.SetStyles(m.styles)
	return m.ds.View(m.width, m.height)
}

// Selected returns the ID of the session the user selected.
func (m SessionListModel) Selected() string {
	return m.selected
}

// Quitting reports whether dismissed.
func (m SessionListModel) Quitting() bool { return m.quitting }
