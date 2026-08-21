package views

import (
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// kuiLogoPairs is the two-sided OpenCode-style █▀▀█ logo rendered as left/right pairs.
// The shadow column (left) is rendered with tint(background, fg, 0.25) and the main column (right)
// uses a theme syntax* derived color (SyntaxKeyword/SyntaxOperator) — never hard-coded hex.
var kuiLogoPairs = [][2]string{
	{"█▀▀█ ", "█  █"},
	{"█  █ ", "█  █"},
	{"█▀▀█ ", "█  █"},
	{"█    ", "█  █"},
	{"█▄▄█ ", "█▄▄█"},
}

// kuiLogo retains the original single-slice form for backward compatibility / height counting.
var kuiLogo = []string{
	"█▀▀█ █  █",
	"█  █ █  █",
	"█▀▀█ █  █",
	"█    █  █",
	"█▄▄█ █▄▄█",
}

// LogoModel renders the ASCII art logo centered within the terminal.
type LogoModel struct {
	styles *theme.Styles
}

// NewLogoModel creates a LogoModel with the given styles.
func NewLogoModel(styles *theme.Styles) LogoModel {
	return LogoModel{styles: styles}
}

// View renders the logo centered within the given width using two-tone tint shadow.
// Shadow is computed as Tint(background, syntaxColor, 0.25) per REQ-TUI-HOME-2.
func (m LogoModel) View(width int) string {
	if m.styles == nil || m.styles.Theme == nil {
		return ""
	}
	t := m.styles.Theme
	bg := t.Background
	if bg == "" {
		bg = t.BG
	}
	fg := t.SyntaxKeyword
	if fg == "" {
		fg = t.SyntaxOperator
	}
	if fg == "" {
		fg = t.Accent
	}
	if fg == "" {
		fg = t.Text
	}
	if fg == "" {
		fg = t.FG
	}
	shadowHex := theme.Tint(bg, fg, 0.25)
	shadowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(shadowHex)).Bold(true)
	mainStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Bold(true)

	var lines []string
	for _, pair := range kuiLogoPairs {
		left := shadowStyle.Render(pair[0])
		right := mainStyle.Render(pair[1])
		lines = append(lines, left+right)
	}
	rendered := strings.Join(lines, "\n")
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, rendered)
}
