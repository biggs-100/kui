package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HeaderModel renders the profile tab bar. It shows one tab per profile
// with the active profile visually marked. When no profiles are provided,
// it renders a hint instead (REQ-TUI-PROF-4).
type HeaderModel struct {
	profiles []string
	active   int
}

// NewHeaderModel creates a HeaderModel with the given profile names and
// active index. An empty or nil profiles slice triggers the no-profiles
// fallback (REQ-TUI-PROF-4).
func NewHeaderModel(profiles []string, active int) HeaderModel {
	return HeaderModel{
		profiles: profiles,
		active:   active,
	}
}

var (
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Padding(0, 1)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Faint(true)
)

// Render produces the full header string. When profiles are present, each
// profile is rendered as a tab; the active one uses bold styling
// (REQ-TUI-PROF-1). When no profiles exist, a hint is rendered
// (REQ-TUI-PROF-4).
func (m HeaderModel) Render() string {
	if len(m.profiles) == 0 {
		return hintStyle.Render("no profiles available")
	}

	var parts []string
	for i, p := range m.profiles {
		label := fmt.Sprintf(" %s ", p)
		if i == m.active {
			parts = append(parts, activeTabStyle.Render(label))
		} else {
			parts = append(parts, inactiveTabStyle.Render(label))
		}
	}
	return strings.Join(parts, " ")
}
