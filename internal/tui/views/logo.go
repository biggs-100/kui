package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/biggs-100/kui/internal/tui/theme"
)

// kuiLogo is the ASCII art for the home screen logo.
var kuiLogo = []string{
	"██╗  ██╗██╗██╗██╗     ██╗",
	"██║ ██╔╝██║██║██║     ██║",
	"█████╔╝ ██║██║██║     ██║",
	"██╔═██╗ ██║██║██║     ██║",
	"██║  ██╗██║██║███████╗██║",
	"╚═╝  ╚═╝╚═╝╚═╝╚══════╝╚═╝",
}

// LogoModel renders the ASCII art logo centered within the terminal.
type LogoModel struct {
	styles *theme.Styles
}

// NewLogoModel creates a LogoModel with the given styles.
func NewLogoModel(styles *theme.Styles) LogoModel {
	return LogoModel{styles: styles}
}

// View renders the logo centered within the given width.
func (m LogoModel) View(width int) string {
	if m.styles == nil {
		return ""
	}

	var lines []string
	for _, line := range kuiLogo {
		lines = append(lines, m.styles.LogoAccent.Render(line))
	}

	rendered := strings.Join(lines, "\n")
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, rendered)
}
