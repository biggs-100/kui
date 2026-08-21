package views

import (
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// HomeView composes the home screen: logo + prompt + footer.
// It uses flex spacers with flexGrow for vertical centering and a height-4 spacer
// between logo and prompt. The centered column uses maxWidth 75 or 70% auto.
type HomeView struct {
	logo   LogoModel
	prompt HomePromptModel
	width  int
	height int
	styles *theme.Styles
	toast  string
}

// NewHomeView creates a HomeView with the given styles and dimensions.
func NewHomeView(styles *theme.Styles, width, height int) HomeView {
	return HomeView{
		logo:   NewLogoModel(styles),
		prompt: NewHomePromptModel(styles),
		width:  width,
		height: height,
		styles: styles,
	}
}

// SetSize updates the terminal dimensions.
func (m *HomeView) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.prompt.SetHeight(height)
}

// SetStyles updates the theme styles for the home view and its sub-views.
func (m *HomeView) SetStyles(s *theme.Styles) {
	m.styles = s
	m.logo = NewLogoModel(s)
	m.prompt.SetStyles(s)
}

// IsZero reports whether the view has been initialized.
func (m HomeView) IsZero() bool {
	return m.styles == nil
}

// SetInput sets the prompt input value.
func (m *HomeView) SetInput(val string) {
	m.prompt.SetValue(val)
}

// GetInput returns the current prompt input value.
func (m HomeView) GetInput() string {
	return m.prompt.Value()
}

// SetToast sets toast content to be rendered inside the centered column.
func (m *HomeView) SetToast(toast string) {
	m.toast = toast
}

// View renders the home screen with flex-spacer vertical centering.
// Layout: flex top spacer (flexGrow) + logo + height-4 spacer + prompt (centered) + toast (inside column) + flex bottom spacer (flexGrow).
func (m HomeView) View() string {
	if m.styles == nil || m.width == 0 || m.height == 0 {
		return ""
	}

	logoStr := m.logo.View(m.width)
	promptStr := m.prompt.View(m.width)

	logoHeight := strings.Count(logoStr, "\n") + 1
	if logoHeight < 1 {
		logoHeight = 5
	}
	promptHeight := strings.Count(promptStr, "\n") + 1
	if promptHeight < 1 {
		promptHeight = 3
	}
	spacerHeight := 4
	toastHeight := 0
	var toastCentered string
	if m.toast != "" {
		toastHeight = strings.Count(m.toast, "\n") + 1
		toastCentered = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, m.toast)
	}

	contentHeight := logoHeight + spacerHeight + promptHeight + toastHeight
	available := m.height - contentHeight
	if available < 0 {
		available = 0
	}
	topPad := available / 2
	bottomPad := available - topPad

	var b strings.Builder

	for i := 0; i < topPad; i++ {
		b.WriteString("\n")
	}

	b.WriteString(logoStr)
	for i := 0; i < spacerHeight; i++ {
		b.WriteString("\n")
	}

	promptCentered := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, promptStr)
	b.WriteString(promptCentered)

	if m.toast != "" {
		b.WriteString("\n")
		b.WriteString(toastCentered)
	}

	for i := 0; i < bottomPad; i++ {
		b.WriteString("\n")
	}

	return b.String()
}
