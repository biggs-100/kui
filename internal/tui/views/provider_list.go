package views

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProviderInfo describes a provider for login/logout selection.
type ProviderInfo struct {
	ID       string
	Name     string
	AuthType string
}

// AvailableProviders returns provider options for /login completions, deduped by ID like Pi getLoginProviderCompletionOptions.
func AvailableProviders() []ProviderInfo {
	return []ProviderInfo{
		{ID: "openai", Name: "OpenAI", AuthType: "API Key"},
		{ID: "anthropic", Name: "Anthropic", AuthType: "OAuth/API Key"},
		{ID: "opencode", Name: "Opencode", AuthType: "API Key"},
		{ID: "opencode-go", Name: "Opencode Go", AuthType: "API Key"},
		{ID: "gemini", Name: "Gemini", AuthType: "API Key"},
	}
}

// providerItem wraps ProviderInfo for the bubbles list.
type providerItem struct {
	info ProviderInfo
}

func (i providerItem) FilterValue() string { return i.info.ID + " " + i.info.Name + " " + i.info.AuthType }

type providerItemDelegate struct{}

func (d providerItemDelegate) Height() int                             { return 1 }
func (d providerItemDelegate) Spacing() int                            { return 0 }
func (d providerItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d providerItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	pi, ok := item.(providerItem)
	if !ok {
		return
	}
	display := fmt.Sprintf("%s · %s", pi.info.Name, pi.info.AuthType)
	if len(display) > 60 {
		display = display[:57] + "..."
	}
	// Show ID as primary
	label := pi.info.ID
	if len(label) > 20 {
		label = label[:17] + "..."
	}
	line := fmt.Sprintf("%-20s %s", label, display)
	if index == m.Index() {
		fmt.Fprintf(w, "▸ %s", line)
	} else {
		fmt.Fprintf(w, "  %s", line)
	}
}

// ProviderListModel wraps a list for interactive provider selection.
type ProviderListModel struct {
	list     list.Model
	infos    []ProviderInfo
	selected string
	quitting bool
	width    int
	height   int
}

// NewProviderListModel creates a ProviderListModel.
func NewProviderListModel(infos []ProviderInfo, width, height int) ProviderListModel {
	items := make([]list.Item, len(infos))
	for i, info := range infos {
		items[i] = providerItem{info: info}
	}
	l := list.New(items, providerItemDelegate{}, width, height)
	l.Title = "Providers"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)
	l.SetShowHelp(false)
	return ProviderListModel{
		list:   l,
		infos:  infos,
		width:  width,
		height: height,
	}
}

// Update handles keyboard for provider list.
func (m ProviderListModel) Update(msg tea.Msg) (ProviderListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if idx := m.list.Index(); idx >= 0 && idx < len(m.infos) {
				m.selected = m.infos[idx].ID
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

// View renders the provider list.
func (m ProviderListModel) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

// Selected returns the provider ID the user selected.
func (m ProviderListModel) Selected() string {
	return m.selected
}

// Quitting reports whether dismissed without selection.
func (m ProviderListModel) Quitting() bool {
	return m.quitting
}
