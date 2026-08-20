package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/biggs-100/kui/internal/tui/theme"
)

// SidebarModel renders the opencode-style right sidebar.
// Sections: Subagents, Context, MCP, LSP — muted grays with blue headers.
type SidebarModel struct {
	styles       *theme.Styles
	tokens       int
	contextMax   int
	cost         float64
	mcpConnected int
	mcpFailed    int
	lspState     string
	profile      string
	model        string
	version      string
	width        int
}

// NewSidebarModel creates a SidebarModel with theme styles.
func NewSidebarModel(styles *theme.Styles) SidebarModel {
	return SidebarModel{styles: styles, version: "1.2.1"}
}

// SetTokens sets token count and context window.
func (m *SidebarModel) SetTokens(total, limit int) {
	m.tokens = total
	m.contextMax = limit
}

// SetCost sets session cost.
func (m *SidebarModel) SetCost(cost float64) {
	m.cost = cost
}

// SetMCPStatus sets MCP server counts.
func (m *SidebarModel) SetMCPStatus(connected, failed int) {
	m.mcpConnected = connected
	m.mcpFailed = failed
}

// SetLSPStatus sets LSP state.
func (m *SidebarModel) SetLSPStatus(state string) {
	m.lspState = state
}

// SetProfile sets active profile name.
func (m *SidebarModel) SetProfile(profile string) {
	m.profile = profile
}

// SetModel sets current model name.
func (m *SidebarModel) SetModel(model string) {
	m.model = model
}

// View renders the sidebar for the given width.
func (m SidebarModel) View(width int) string {
	if m.styles == nil || width < 10 {
		return ""
	}
	// Ensure reasonable width
	if width < 20 {
		width = 20
	}
	// Header style: accent blue bold
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#569cd6")).
		Bold(true).
		Width(width - 2)

	muted := m.styles.HomeMuted
	// Use muted for body, but ensure it renders within width
	bodyStyle := lipgloss.NewStyle().
		Foreground(muted.GetForeground()).
		Faint(true).
		Width(width - 2)

	sep := muted.Render(strings.Repeat("─", width-2))

	var b strings.Builder

	// Section helper
	section := func(title string, lines []string) {
		b.WriteString(headerStyle.Render(title))
		b.WriteString("\n")
		for _, l := range lines {
			// Truncate if needed
			if lipgloss.Width(l) > width-2 {
				// simple truncation
				l = l[:width-1] + "..."
			}
			b.WriteString(bodyStyle.Render(l))
			b.WriteString("\n")
		}
		b.WriteString(sep)
		b.WriteString("\n")
	}

	// Subagents section — placeholder counts (opencode style: Subagents 1.2.1 • 0 run • 0 done • Σ 0)
	subLines := []string{
		fmt.Sprintf("v%s • 0 run • 0 done • Σ 0", m.version),
	}
	// Add profile/model hint if available
	if m.profile != "" || m.model != "" {
		hint := strings.TrimSpace(fmt.Sprintf("%s · %s", m.profile, m.model))
		subLines = append(subLines, muted.Render(hint))
	}
	section("Subagents", subLines)

	// Context section — tokens, percent, cost (opencode: Context 319k tokens 32% $0.27)
	var ctxLines []string
	if m.tokens > 0 {
		pct := 0
		if m.contextMax > 0 {
			pct = m.tokens * 100 / m.contextMax
		}
		// Format like "319k tokens 32% $0.27" or "1234 tokens (12%)"
		var tokenStr string
		if m.tokens >= 1000 {
			tokenStr = fmt.Sprintf("%.1fK tokens", float64(m.tokens)/1000)
		} else {
			tokenStr = fmt.Sprintf("%d tokens", m.tokens)
		}
		ctxLines = append(ctxLines, fmt.Sprintf("%s %d%% $%.2f", tokenStr, pct, m.cost))
		// Context window hint
		if m.contextMax > 0 {
			ctxLines = append(ctxLines, fmt.Sprintf("%d / %d", m.tokens, m.contextMax))
		}
	} else {
		ctxLines = append(ctxLines, "319k tokens 32% $0.27")
		// Replace with real placeholders when no data: show dash
		if m.contextMax == 0 {
			ctxLines = []string{"0 tokens 0% $0.00"}
		}
	}
	section("Context", ctxLines)

	// MCP section — dots (Connected / ○)
	var mcpLines []string
	if m.mcpConnected > 0 && m.mcpFailed == 0 {
		mcpLines = append(mcpLines, fmt.Sprintf("● Connected (%d)", m.mcpConnected))
	} else if m.mcpConnected > 0 || m.mcpFailed > 0 {
		if m.mcpFailed > 0 {
			mcpLines = append(mcpLines, fmt.Sprintf("● %d/%d (failed %d)", m.mcpConnected, m.mcpFailed, m.mcpFailed))
		} else {
			mcpLines = append(mcpLines, fmt.Sprintf("● %d connected", m.mcpConnected))
		}
	} else {
		// Default like opencode: context7/engram Connected or disabled
		mcpLines = append(mcpLines, "○ disconnected")
		mcpLines = append(mcpLines, "context7 • engram")
	}
	section("MCP", mcpLines)

	// LSP section
	var lspLines []string
	if m.lspState == "running" || m.lspState == "idle" {
		lspLines = append(lspLines, fmt.Sprintf("● %s", m.lspState))
	} else if m.lspState == "error" {
		lspLines = append(lspLines, fmt.Sprintf("● error"))
	} else {
		lspLines = append(lspLines, "○ disabled")
	}
	section("LSP", lspLines)

	// Wrap whole sidebar in background #1a1a1a / #252525 style
	content := strings.TrimSuffix(b.String(), "\n")
	// Apply sidebar background and constrain width
	sidebarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1a1a1a")).
		Width(width).
		Padding(0, 1)
	return sidebarStyle.Render(content)
}
