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

// View renders the logo centered within the given width using the upstream
// two-tone treatment: left half in textMuted regular weight, right half in
// text bold — a monochrome mark so colorful conversation content carries all
// the chroma (REQ-TUI-HOME-2).
func (m LogoModel) View(width int) string {
	if m.styles == nil || m.styles.Theme == nil {
		return ""
	}
	t := m.styles.Theme
	mainFg := t.Text
	if mainFg == "" {
		mainFg = t.FG
	}
	shadowFg := t.TextMuted
	if shadowFg == "" {
		shadowFg = t.Hint
	}
	if shadowFg == "" {
		shadowFg = mainFg
	}
	shadowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(shadowFg))
	mainStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(mainFg)).Bold(true)

	var lines []string
	for _, pair := range kuiLogoPairs {
		left := shadowStyle.Render(pair[0])
		right := mainStyle.Render(pair[1])
		lines = append(lines, left+right)
	}
	rendered := strings.Join(lines, "\n")
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, rendered)
}
