package views

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/ui"
	"github.com/charmbracelet/lipgloss"
)

// placeholderPool provides varied placeholders (not single "Ask kui...") per REQ-TUI-HOME-3.
var placeholderPool = []string{
	"Ask anything...",
	"Plan, build, ship...",
	"What do you want to build?",
	"Describe your idea...",
	"Ask kui to help...",
	"How can kui help?",
}

var placeholderCounter uint64

// ResetPlaceholderCounter resets the placeholder rotation (for deterministic golden tests).
func ResetPlaceholderCounter() {
	atomic.StoreUint64(&placeholderCounter, 0)
}

// HomePromptModel renders a bordered text input for the home screen.
// It wraps user input with SplitBorder and shows a placeholder when empty.
// It supports "!" shell mode at offset 0 and extmarks virtual text for
// ● [File]/[Image]/[Pasted ~N lines] as muted NotAvailable when store absent.
type HomePromptModel struct {
	value       string
	placeholder string
	styles      *theme.Styles
	termHeight  int
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

// SetHeight sets the terminal height for MaxHeight calc (max(6, height/3)).
func (m *HomePromptModel) SetHeight(h int) {
	m.termHeight = h
}

// MaxHeight returns max(6, termHeight/3) per REQ-TUI-HOME-3.
func (m HomePromptModel) MaxHeight() int {
	h := m.termHeight / 3
	if h < 6 {
		h = 6
	}
	return h
}

// View renders the prompt with SplitBorder+EmptyBorder decorative bottom ▀.
// Width is 70% of terminal width capped at 75 per REQ-TUI-HOME-3.
func (m HomePromptModel) View(width int) string {
	if m.styles == nil {
		return ""
	}
	// OpenCode style: prompt width is 70% of terminal width, capped at 75.
	promptWidth := width * 70 / 100
	if promptWidth > 75 {
		promptWidth = 75
	}
	if promptWidth < 20 {
		promptWidth = 20
	}
	if promptWidth > width-4 {
		promptWidth = width - 4
	}

	// Resolve theme for backgroundElement and border colors.
	var bgElement, borderColor string
	if m.styles.Theme != nil {
		bgElement = m.styles.Theme.BackgroundElement
		if bgElement == "" {
			bgElement = m.styles.Theme.BGHighlight
		}
		borderColor = m.styles.Theme.Border
		if borderColor == "" {
			borderColor = m.styles.Theme.BorderSubtle
		}
	}
	// Shell mode at offset 0 triggers distinct style (Warning/Primary accent).
	isShell := strings.HasPrefix(strings.TrimSpace(m.value), "!")
	if isShell && m.styles.Theme != nil {
		if m.styles.Theme.Warning != "" {
			borderColor = m.styles.Theme.Warning
		} else if m.styles.Theme.Primary != "" {
			borderColor = m.styles.Theme.Primary
		}
	}

	borderStyle := lipgloss.NewStyle().
		Border(ui.SplitBorder).
		BorderForeground(lipgloss.Color(borderColor)).
		Background(lipgloss.Color(bgElement)).
		Padding(0, 1).
		Width(promptWidth)

	cursor := m.styles.LogoAccent.Render("▏")
	var text string
	if m.value == "" {
		poolIdx := int(atomic.AddUint64(&placeholderCounter, 1)-1) % len(placeholderPool)
		ph := placeholderPool[poolIdx]
		placeholder := lipgloss.NewStyle().
			Foreground(m.styles.HomeMuted.GetForeground()).
			Faint(true).
			Render(ph)
		text = placeholder + cursor
	} else {
		// Shell mode indicator already in value ("!"); keep it visible.
		text = m.value + cursor
		// Extmarks virtual text for ● [File]/[Image]/[Pasted ~N lines] as muted NotAvailable.
		if ext := m.extmarkText(); ext != "" {
			text = text + ext
		}
	}

	content := borderStyle.Render(text)
	// Decorative bottom ▀ (EmptyBorder style) spans promptWidth+2 (border padding).
	decorative := lipgloss.NewStyle().
		Foreground(lipgloss.Color(bgElement)).
		Render(strings.Repeat(ui.PromptBottom, promptWidth+2))

	return content + "\n" + decorative
}

func (m HomePromptModel) extmarkText() string {
	if m.styles == nil {
		return ""
	}
	v := strings.ToLower(m.value)
	var marks []string
	if strings.Contains(v, "file") || strings.Contains(m.value, "@") || strings.Contains(m.value, "[File]") {
		marks = append(marks, "● [File]")
	}
	if strings.Contains(v, "image") || strings.Contains(m.value, "[Image]") {
		marks = append(marks, "● [Image]")
	}
	if strings.Contains(v, "paste") || strings.Contains(m.value, "[Pasted") {
		lines := strings.Count(m.value, "\n") + 1
		if lines == 1 {
			lines = 5
		}
		marks = append(marks, fmt.Sprintf("● [Pasted ~%d lines]", lines))
	}
	if len(marks) == 0 {
		return ""
	}
	return m.styles.HomeMuted.Render(" " + strings.Join(marks, " "))
}
