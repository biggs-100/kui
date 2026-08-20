package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// SidebarModel renders the opencode-style right sidebar.
// Only real, controller-backed fields are shown: token usage/cost (Context)
// and the active profile/model (Session). kui tracks no MCP/LSP server state
// and has no version string, so those sections are intentionally omitted.
type SidebarModel struct {
	styles     *theme.Styles
	tokens     int
	contextMax int
	cost       float64
	profile    string
	model      string
	width      int
}

// NewSidebarModel creates a SidebarModel with theme styles.
func NewSidebarModel(styles *theme.Styles) SidebarModel {
	return SidebarModel{styles: styles}
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
	if width < 20 {
		width = 20
	}

	// Header style: accent blue bold (theme token, not a literal)
	headerStyle := m.styles.LogoAccent

	muted := m.styles.HomeMuted
	bodyStyle := lipgloss.NewStyle().
		Foreground(muted.GetForeground()).
		Faint(true).
		Width(width - 2)

	sep := muted.Render(strings.Repeat("─", width-2))

	var b strings.Builder

	section := func(title string, lines []string) {
		b.WriteString(headerStyle.Render(title))
		b.WriteString("\n")
		for _, l := range lines {
			if lipgloss.Width(l) > width-2 {
				l = l[:width-1] + "..."
			}
			b.WriteString(bodyStyle.Render(l))
			b.WriteString("\n")
		}
		b.WriteString(sep)
		b.WriteString("\n")
	}

	// Context section — real tokens, percent, cost from the controller.
	var ctxLines []string
	if m.tokens > 0 {
		pct := 0
		if m.contextMax > 0 {
			pct = m.tokens * 100 / m.contextMax
		}
		ctxLines = append(ctxLines, fmt.Sprintf("%d tokens %d%% $%.2f", m.tokens, pct, m.cost))
		if m.contextMax > 0 {
			ctxLines = append(ctxLines, fmt.Sprintf("%d / %d", m.tokens, m.contextMax))
		}
	} else {
		ctxLines = append(ctxLines, "0 tokens 0% $0.00")
	}
	section("Context", ctxLines)

	// Session section — real profile/model only (never fabricated).
	if m.profile != "" || m.model != "" {
		var sessLines []string
		if m.profile != "" {
			sessLines = append(sessLines, "profile  "+m.profile)
		}
		if m.model != "" {
			sessLines = append(sessLines, "model  "+m.model)
		}
		section("Session", sessLines)
	}

	content := strings.TrimSuffix(b.String(), "\n")
	// Use the theme's sidebar style (background BGSidebar + padding) instead of
	// a hardcoded hex literal.
	return m.styles.Sidebar.Width(width).Render(content)
}
