package views

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/biggs-100/kui/internal/tui/theme"
)

// HomePromptModel renders a bordered text input for the home screen.
// It wraps user input with a rounded border and shows a placeholder when empty.
type HomePromptModel struct {
	value       string
	placeholder string
	styles      *theme.Styles
}

// NewHomePromptModel creates a HomePromptModel.
func NewHomePromptModel(styles *theme.Styles) HomePromptModel {
	return HomePromptModel{
		placeholder: "Ask kui...",
		styles:      styles,
	}
}

// SetValue sets the current input value.
func (m *HomePromptModel) SetValue(val string) {
	m.value = val
}

// Value returns the current input value.
func (m HomePromptModel) Value() string {
	return m.value
}

// Clear resets the input value.
func (m *HomePromptModel) Clear() {
	m.value = ""
}

// Submit returns the current value and clears it.
func (m *HomePromptModel) Submit() string {
	val := m.value
	m.value = ""
	return val
}

// SetStyles updates the theme styles for the prompt.
func (m *HomePromptModel) SetStyles(s *theme.Styles) {
	m.styles = s
}

// View renders the prompt with a rounded border and a cursor indicator.
func (m HomePromptModel) View(width int) string {
	if m.styles == nil {
		return ""
	}

	// OpenCode style: prompt is ~60-70 chars wide, centered, with rounded border.
	promptWidth := width - 20
	if promptWidth > 70 {
		promptWidth = 70
	}
	if promptWidth < 20 {
		promptWidth = 20
	}
	if promptWidth > width-4 {
		promptWidth = width - 4
	}

	border := lipgloss.RoundedBorder()
	borderStyle := lipgloss.NewStyle().
		Border(border).
		BorderForeground(m.styles.HomeBorder.GetBorderTopForeground()).
		Padding(0, 1).
		Width(promptWidth)

	cursor := m.styles.LogoAccent.Render("▏")
	var text string
	if m.value == "" {
		placeholder := lipgloss.NewStyle().
			Foreground(m.styles.HomeMuted.GetForeground()).
			Faint(true).
			Render(m.placeholder)
		text = placeholder + cursor
	} else {
		text = m.value + cursor
	}

	return borderStyle.Render(text)
}
