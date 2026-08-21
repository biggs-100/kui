package views

import (
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// HomeFooterModel renders the grounding footer for the home screen,
// mirroring feature-plugins/home/footer.tsx: a space-between row with the
// working directory in textMuted on the left and "• kui <version>" (success
// dot, bold text name, muted version) on the right. The home_bottom plugin
// slot replaces the row content when present.
type HomeFooterModel struct {
	styles        *theme.Styles
	dir           string
	width         int
	pluginContent string
	// Retained for compatibility but not rendered as fabricated LSP/MCP.
	lspConnected bool
	mcpConnected bool
}

// NewHomeFooterModel creates a HomeFooterModel.
func NewHomeFooterModel(styles *theme.Styles, dir string) HomeFooterModel {
	return HomeFooterModel{
		styles: styles,
		dir:    dir,
	}
}

// SetWidth sets the row width used for the space-between composition.
func (m *HomeFooterModel) SetWidth(w int) {
	m.width = w
}

// SetLSPConnected retains compatibility but does not fabricate LSP display (home is empty).
func (m *HomeFooterModel) SetLSPConnected(connected bool) {
	m.lspConnected = connected
}

// SetMCPConnected retains compatibility but does not fabricate MCP display.
func (m *HomeFooterModel) SetMCPConnected(connected bool) {
	m.mcpConnected = connected
}

// SetPluginContent sets the home_bottom plugin slot content. When set, it
// replaces the standard dir/version row.
func (m *HomeFooterModel) SetPluginContent(content string) {
	m.pluginContent = content
}

// Render produces the home footer string: dir on the left, version badge on
// the right, or the plugin slot content when present.
func (m HomeFooterModel) Render() string {
	if m.styles == nil {
		return ""
	}
	if m.pluginContent != "" {
		return m.styles.HomeMuted.Render(m.pluginContent)
	}
	left := ""
	if m.dir != "" {
		left = m.styles.HomeMuted.Render(m.dir)
	}
	right := ""
	if m.styles.Theme != nil {
		if ver := getVersion(); ver != "" {
			dot := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.Success)).Render("•")
			name := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.styles.Theme.Text)).Render("kui")
			right = dot + " " + name + " " + m.styles.HomeMuted.Render(ver)
		}
	}
	if left == "" && right == "" {
		return ""
	}
	return joinSpaceBetween(left, right, m.width)
}
