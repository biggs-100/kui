package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
)

// FooterModel renders the status bar at the bottom of the screen.
// Only real, controller-backed fields are shown: directory, token usage,
// session cost, and the current model. kui has no version string and tracks
// no MCP/LSP server state, so those are intentionally omitted (no fabrication).
type FooterModel struct {
	dir        string
	model      string
	tokens     int
	contextMax int
	cost       float64
	styles     *theme.Styles
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

// Render produces the footer string.
// Format: <dir> • <tokens> (<pct>%) · $<cost> • <model> • ctrl+p commands
func (m FooterModel) Render() string {
	if m.styles == nil {
		return ""
	}

	dash := "—"
	sep := m.styles.HomeMuted.Render(" • ")
	innerSep := m.styles.HomeMuted.Render(" · ")

	// Directory — muted faint (left)
	dir := m.dir
	if dir == "" {
		dir = dash
	}
	dirStr := m.styles.HomeMuted.Render(dir)

	// Tokens — real controller value; hidden (dash) when zero
	var tokensStr string
	if m.tokens > 0 {
		pct := 0
		if m.contextMax > 0 {
			pct = m.tokens * 100 / m.contextMax
		}
		tokensStr = m.styles.HomeMuted.Render(fmt.Sprintf("%d tokens (%d%%)", m.tokens, pct))
	} else {
		tokensStr = m.styles.HomeMuted.Render(dash + " tokens")
	}

	// Cost — real accumulated session cost
	costStr := m.styles.HomeMuted.Render(fmt.Sprintf("$%.2f", m.cost))

	centerStr := tokensStr + innerSep + costStr

	// Model — real current model (kept subtle; OpenCode shows it in the input bar)
	model := m.model
	if model == "" {
		model = dash
	}
	modelStr := m.styles.HomeMuted.Render(model)

	// Command palette hint — ctrl+p is a real binding (app.go KeyCtrlP)
	ctrlPStr := m.styles.HomeMuted.Render("ctrl+p commands")

	parts := []string{dirStr, centerStr, modelStr, ctrlPStr}
	return strings.Join(parts, sep)
}
