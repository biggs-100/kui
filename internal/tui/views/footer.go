package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
)

// FooterModel renders the status bar at the bottom of the screen.
type FooterModel struct {
	dir          string
	model        string
	tokens       int
	contextMax   int
	cost         float64
	mcpConnected int
	mcpFailed    int
	styles       *theme.Styles
}

// NewFooterModel creates a FooterModel.
func NewFooterModel(styles *theme.Styles) FooterModel {
	return FooterModel{styles: styles}
}

// SetDir sets the working directory.
func (m *FooterModel) SetDir(dir string) {
	m.dir = dir
}

// SetModel sets the current model name.
func (m *FooterModel) SetModel(model string) {
	m.model = model
}

// SetTokens sets token count and context limit.
func (m *FooterModel) SetTokens(total, limit int) {
	m.tokens = total
	m.contextMax = limit
}

// SetCost sets the session cost.
func (m *FooterModel) SetCost(cost float64) {
	m.cost = cost
}

// SetMCPStatus sets MCP server counts.
func (m *FooterModel) SetMCPStatus(connected, failed int) {
	m.mcpConnected = connected
	m.mcpFailed = failed
}

// Render produces the footer string.
func (m FooterModel) Render() string {
	if m.styles == nil {
		return ""
	}

	dash := "—"

	// Directory
	dir := m.dir
	if dir == "" {
		dir = dash
	}

	// Model
	model := m.model
	if model == "" {
		model = dash
	}

	// Tokens
	var tokens string
	if m.tokens > 0 {
		pct := 0
		if m.contextMax > 0 {
			pct = m.tokens * 100 / m.contextMax
		}
		tokens = fmt.Sprintf("%d tokens (%d%%)", m.tokens, pct)
	} else {
		tokens = dash + " tokens"
	}

	// Cost
	cost := fmt.Sprintf("$%.2f", m.cost)

	// MCP status
	mcp := fmt.Sprintf("MCP: %d connected", m.mcpConnected)
	if m.mcpFailed > 0 {
		mcp = fmt.Sprintf("MCP: %d/%d", m.mcpConnected, m.mcpFailed)
	}

	parts := []string{dir, model, tokens, cost, mcp}
	return m.styles.StatusLine.Render(strings.Join(parts, " | "))
}
