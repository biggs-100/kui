package views

import (
	"github.com/biggs-100/kui/internal/tui/theme"
)

// HomeFooterModel renders the minimal footer for the home screen.
// It is empty plus home_bottom plugin slot (muted NotAvailable when absent).
// It MUST NOT show fabricated dir • LSP ○/● • MCP ○/● invention.
// If backing sync.data.lsp/mcp absent, footer omits counts as muted.
type HomeFooterModel struct {
	styles        *theme.Styles
	dir           string
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

// SetLSPConnected retains compatibility but does not fabricate LSP display (home is empty).
func (m *HomeFooterModel) SetLSPConnected(connected bool) {
	m.lspConnected = connected
}

// SetMCPConnected retains compatibility but does not fabricate MCP display.
func (m *HomeFooterModel) SetMCPConnected(connected bool) {
	m.mcpConnected = connected
}

// SetPluginContent sets the home_bottom plugin slot content. When empty, footer is muted placeholder.
func (m *HomeFooterModel) SetPluginContent(content string) {
	m.pluginContent = content
}

// Render produces the minimal home footer string.
// When no plugin slot and no sync data, it returns empty or muted placeholder, not "• LSP".
func (m HomeFooterModel) Render() string {
	if m.styles == nil {
		return ""
	}
	if m.pluginContent != "" {
		return m.styles.HomeMuted.Render(m.pluginContent)
	}
	// No plugin slot and no sync data → empty or muted placeholder (NotAvailable)
	// Return empty to satisfy "empty plus plugin slot (muted NotAvailable when absent)"
	// We return a faint dash as muted placeholder to be visible but not fabricated.
	return ""
}
