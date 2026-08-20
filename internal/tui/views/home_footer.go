package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
)

// HomeFooterModel renders the minimal footer for the home screen.
// Format: directory • LSP ● • MCP ● • /status
type HomeFooterModel struct {
	dir         string
	lspConnected bool
	mcpConnected bool
	styles      *theme.Styles
}

// NewHomeFooterModel creates a HomeFooterModel.
func NewHomeFooterModel(styles *theme.Styles, dir string) HomeFooterModel {
	return HomeFooterModel{
		styles: styles,
		dir:    dir,
	}
}

// SetLSPConnected sets whether the LSP server is connected.
func (m *HomeFooterModel) SetLSPConnected(connected bool) {
	m.lspConnected = connected
}

// SetMCPConnected sets whether the MCP server is connected.
func (m *HomeFooterModel) SetMCPConnected(connected bool) {
	m.mcpConnected = connected
}

// Render produces the minimal home footer string.
func (m HomeFooterModel) Render() string {
	if m.styles == nil {
		return ""
	}

	sep := m.styles.HomeMuted.Render(" • ")

	// Directory
	dir := m.dir
	if dir == "" {
		dir = "—"
	}
	dirStr := m.styles.HomeMuted.Render(dir)

	// LSP dot
	lspDot := "●"
	lspStyle := m.styles.StatusOK
	if !m.lspConnected {
		lspDot = "○"
		lspStyle = m.styles.HomeMuted
	}
	lspStr := fmt.Sprintf("%s %s", m.styles.HomeMuted.Render("LSP"), lspStyle.Render(lspDot))

	// MCP dot
	mcpDot := "●"
	mcpStyle := m.styles.StatusOK
	if !m.mcpConnected {
		mcpDot = "○"
		mcpStyle = m.styles.HomeMuted
	}
	mcpStr := fmt.Sprintf("%s %s", m.styles.HomeMuted.Render("MCP"), mcpStyle.Render(mcpDot))

	// /status
	statusStr := m.styles.HomeMuted.Render("/status")

	parts := []string{dirStr, lspStr, mcpStr, statusStr}
	return m.styles.StatusLine.Render(strings.Join(parts, sep))
}
