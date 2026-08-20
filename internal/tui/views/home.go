package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/biggs-100/kui/internal/tui/theme"
)

// HomeView composes the home screen: logo + prompt + footer.
// It vertically centers the logo and prompt, and places the footer at the bottom.
type HomeView struct {
	logo   LogoModel
	prompt HomePromptModel
	width  int
	height int
	styles *theme.Styles
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

// View renders the home screen with vertically centered logo and prompt.
func (m HomeView) View() string {
	if m.styles == nil || m.width == 0 || m.height == 0 {
		return ""
	}

	logoStr := m.logo.View(m.width)
	promptStr := m.prompt.View(m.width)

	// Count lines in logo to calculate vertical centering
	logoLines := strings.Split(logoStr, "\n")
	logoHeight := len(logoLines)
	promptHeight := 3 // bordered prompt is 3 lines (top border + content + bottom border)

	// Reserve space for footer (1 line) + spacing, so centering doesn't push footer off-screen.
	footerReserve := 2
	effectiveHeight := m.height - footerReserve
	if effectiveHeight < 1 {
		effectiveHeight = m.height
	}

	// Calculate spacing: center the logo+prompt block vertically within effective height.
	totalContent := logoHeight + 2 + promptHeight // logo + 2 spacers + prompt
	topPad := (effectiveHeight - totalContent) / 2
	if topPad < 1 {
		topPad = 1
	}

	var b strings.Builder

	// Top padding
	for i := 0; i < topPad; i++ {
		b.WriteString("\n")
	}

	// Logo
	b.WriteString(logoStr)
	b.WriteString("\n\n") // spacer between logo and prompt

	// Prompt (centered)
	promptCentered := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, promptStr)
	b.WriteString(promptCentered)

	return b.String()
}
