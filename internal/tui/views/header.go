package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
)

// HeaderModel renders the profile tab bar. It shows one tab per profile
// with the active profile visually marked. When no profiles are provided,
// it renders a hint instead (REQ-TUI-PROF-4).
type HeaderModel struct {
	profiles []string
	active   int
	styles   *theme.Styles
}

// NewHeaderModel creates a HeaderModel with the given profile names and
// active index. An empty or nil profiles slice triggers the no-profiles
// fallback (REQ-TUI-PROF-4).
func NewHeaderModel(profiles []string, active int, styles *theme.Styles) HeaderModel {
	return HeaderModel{
		profiles: profiles,
		active:   active,
		styles:   styles,
	}
}

// Render produces the full header string. When profiles are present, each
// profile is rendered as a tab; the active one uses bold styling
// (REQ-TUI-PROF-1). When no profiles exist, a hint is rendered
// (REQ-TUI-PROF-4).
func (m HeaderModel) Render() string {
	if len(m.profiles) == 0 {
		return m.styles.Hint.Render("no profiles available")
	}

	var parts []string
	for i, p := range m.profiles {
		label := fmt.Sprintf(" %s ", p)
		if i == m.active {
			parts = append(parts, m.styles.ActiveTab.Render(label))
		} else {
			parts = append(parts, m.styles.InactiveTab.Render(label))
		}
	}
	return strings.Join(parts, " ")
}
