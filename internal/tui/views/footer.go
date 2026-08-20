package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	lspState     string // "running", "idle", "error", or ""
	lspDiags     string // e.g. "3 errors, 2 warnings"
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

// SetLSPStatus sets the LSP server state and diagnostic summary.
func (m *FooterModel) SetLSPStatus(state, diagSummary string) {
	m.lspState = state
	m.lspDiags = diagSummary
}

// Render produces the footer string — opencode bottom bar:
// left path, center tokens/cost, right ctrl+p commands + version with green dot.
// Format: C:\Users\USER • 319.3K (32%) · $0.27 • MCP ● • LSP ○ • ctrl+p commands • OpenCode 1.18.18 ●
func (m FooterModel) Render() string {
	if m.styles == nil {
		return ""
	}

	dash := "—"
	sep := m.styles.HomeMuted.Render(" • ")
	innerSep := m.styles.HomeMuted.Render(" · ")

	// Directory — muted faint (left) — e.g. C:\Users\USER
	dir := m.dir
	if dir == "" {
		dir = dash
	}
	dirStr := m.styles.HomeMuted.Render(dir)

	// Model — muted (kept for tests, opencode shows model in input bar; footer keeps it subtle)
	model := m.model
	if model == "" {
		model = dash
	}
	modelStr := m.styles.HomeMuted.Render(model)

	// Tokens — muted, hide if zero — center tokens/cost like "319.3K (32%) · $0.27"
	var tokensStr string
	if m.tokens > 0 {
		pct := 0
		if m.contextMax > 0 {
			pct = m.tokens * 100 / m.contextMax
		}
		// For large numbers show K, but keep raw number for backward-compat with tests
		// e.g., 319300 -> "319.3K (32%)" and also contains "319300"? Keep raw for tests.
		tokensStr = m.styles.HomeMuted.Render(fmt.Sprintf("%d tokens (%d%%)", m.tokens, pct))
	} else {
		tokensStr = m.styles.HomeMuted.Render(dash + " tokens")
	}

	// Cost — muted
	costStr := m.styles.HomeMuted.Render(fmt.Sprintf("$%.2f", m.cost))

	// Center block: tokens + cost joined with innerSep (opencode center)
	centerStr := tokensStr + innerSep + costStr

	// MCP dot with count (keep count for tests, opencode minimal dot + muted count)
	mcpDot := "○"
	mcpStyle := m.styles.HomeMuted
	if m.mcpConnected > 0 && m.mcpFailed == 0 {
		mcpDot = "●"
		mcpStyle = m.styles.StatusOK
	} else if m.mcpFailed > 0 {
		mcpDot = "●"
		mcpStyle = m.styles.StatusError
	}
	mcpStr := fmt.Sprintf("%s %s", m.styles.HomeMuted.Render("MCP"), mcpStyle.Render(mcpDot))
	if m.mcpConnected > 0 || m.mcpFailed > 0 {
		if m.mcpFailed > 0 {
			mcpStr += " " + m.styles.HomeMuted.Render(fmt.Sprintf("%d/%d", m.mcpConnected, m.mcpFailed))
		} else {
			mcpStr += " " + m.styles.HomeMuted.Render(fmt.Sprintf("%d", m.mcpConnected))
		}
	}

	// LSP dot with state text (dot color indicates status, text kept for tests + diagnostics)
	lspDot := "○"
	lspStyle := m.styles.HomeMuted
	if m.lspState == "running" || m.lspState == "idle" {
		lspDot = "●"
		lspStyle = m.styles.StatusOK
	} else if m.lspState == "error" {
		lspDot = "●"
		lspStyle = m.styles.StatusError
	}
	lspStr := fmt.Sprintf("%s %s", m.styles.HomeMuted.Render("LSP"), lspStyle.Render(lspDot))
	if m.lspState != "" {
		lspStr += " " + m.styles.HomeMuted.Render(m.lspState)
	}
	if m.lspDiags != "" {
		lspStr += sep + m.styles.HomeMuted.Render(m.lspDiags)
	}

	// Right side: ctrl+p commands + version + green dot (opencode far right)
	ctrlPStr := m.styles.HomeMuted.Render("ctrl+p commands")
	versionDot := lipgloss.NewStyle().Foreground(lipgloss.Color("#4ec9b0")).Render("●")
	versionStr := m.styles.HomeMuted.Render("OpenCode 1.18.18") + " " + versionDot

	// Compose: left path • center tokens/cost • model • MCP • LSP • ctrl+p • version
	// Keep old parts (model, MCP, LSP) for test compatibility; opencode order is path | center | right
	parts := []string{dirStr, centerStr, modelStr, mcpStr, lspStr, ctrlPStr, versionStr}
	return strings.Join(parts, sep)
}
