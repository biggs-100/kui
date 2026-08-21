package views

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/util"
	"github.com/charmbracelet/lipgloss"
)

// SidebarModel renders the opencode-style right sidebar.
// Width MUST be 42 cols (REQ-TUI-APP-2). Uses locale FormatNumber, header
// title+sessionID+workspace, footer version via buildinfo.
type SidebarModel struct {
	styles     *theme.Styles
	tokens     int
	contextMax int
	cost       float64
	profile    string
	model      string
	width      int
	title      string
	sessionID  string
	workspace  string
}

// NewSidebarModel creates a SidebarModel with theme styles.
func NewSidebarModel(styles *theme.Styles) SidebarModel {
	return SidebarModel{styles: styles, width: 42}
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

// SetTitle sets header title (window title / session name).
func (m *SidebarModel) SetTitle(title string) { m.title = title }

// SetSessionID sets session ID shown in header.
func (m *SidebarModel) SetSessionID(id string) { m.sessionID = id }

// SetWorkspace sets workspace path displayed in the bottom-pinned footer.
func (m *SidebarModel) SetWorkspace(ws string) { m.workspace = ws }

// SetWidth sets sidebar width (should be 42 per spec).
func (m *SidebarModel) SetWidth(w int) { m.width = w }

// getVersion returns InstallationVersion via buildinfo if present else empty.
func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		v := info.Main.Version
		if v != "" && v != "(devel)" {
			return v
		}
	}
	return ""
}

// View renders the sidebar for the given width.
func (m SidebarModel) View(width int) string {
	if m.styles == nil || width < 10 {
		return ""
	}
	if width < 20 {
		width = 20
	}
	if m.width == 42 && width != 42 {
		// enforce 42 when wide per spec, but respect passed width if narrow overlay
		if width > 42 {
			width = 42
		}
	}

	content := m.viewBody(width)
	if footer := m.footerLines(width); len(footer) > 0 {
		content = content + "\n" + strings.Join(footer, "\n")
	}
	// Use the theme's sidebar style (background BGSidebar + padding) instead of
	// a hardcoded hex literal.
	return m.styles.Sidebar.Width(width).Render(content)
}

// ViewFullHeight renders the sidebar stretched to exactly height rows: the
// section blocks stay pinned at the top, blank filler fills the middle, and
// the footer lines (workspace path above version line) are pinned at the very
// bottom. The whole block is wrapped once in the Sidebar style so every row —
// including filler — carries the rail background and spans exactly width
// columns (lipgloss styles its width-fill with the style background). When
// the content is taller than height the block renders at natural height.
func (m SidebarModel) ViewFullHeight(width, height int) string {
	if m.styles == nil || width < 10 {
		return ""
	}
	if height < 1 {
		return m.View(width)
	}
	body := m.viewBody(width)
	footer := m.footerLines(width)
	fill := height - len(strings.Split(body, "\n")) - len(footer)

	var b strings.Builder
	b.WriteString(body)
	for i := 0; i < fill; i++ {
		b.WriteString("\n")
	}
	for _, l := range footer {
		b.WriteString("\n")
		b.WriteString(l)
	}
	return m.styles.Sidebar.Width(width).Render(b.String())
}

// viewBody renders the sidebar sections without footer and without the outer
// Sidebar style wrap.
func (m SidebarModel) viewBody(width int) string {
	if width < 20 {
		width = 20
	}
	if m.width == 42 && width != 42 {
		if width > 42 {
			width = 42
		}
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
				l = truncateSidebarLine(l, width-2)
			}
			b.WriteString(bodyStyle.Render(l))
			b.WriteString("\n")
		}
		b.WriteString(sep)
		b.WriteString("\n")
	}

	// Session section: title + sessionID + profile/model (REQ-TUI-APP-2,
	// REQ-TUI-CHAT-6). The workspace path lives in the bottom-pinned footer.
	var sessLines []string
	if m.title != "" {
		sessLines = append(sessLines, m.title)
	}
	if m.sessionID != "" {
		sessLines = append(sessLines, "session "+m.sessionID)
	}
	if m.profile != "" {
		sessLines = append(sessLines, "profile  "+m.profile)
	}
	if m.model != "" {
		sessLines = append(sessLines, "model  "+m.model)
	}
	if len(sessLines) > 0 {
		section("Session", sessLines)
	}

	// Context section — real tokens, percent, cost from the controller via locale.
	var ctxLines []string
	if m.tokens > 0 {
		pct := 0
		if m.contextMax > 0 {
			pct = m.tokens * 100 / m.contextMax
		}
		tokensStr := util.FormatNumber(m.tokens)
		costStr := util.FormatMoney(m.cost)
		ctxLines = append(ctxLines, fmt.Sprintf("%s tokens %d%% %s", tokensStr, pct, costStr))
		if m.contextMax > 0 {
			ctxLines = append(ctxLines, fmt.Sprintf("%s / %s", util.FormatNumber(m.tokens), util.FormatNumber(m.contextMax)))
		}
	} else {
		ctxLines = append(ctxLines, "0 tokens 0% $0.00")
	}
	section("Context", ctxLines)

	return strings.TrimSuffix(b.String(), "\n")
}

// footerLines returns the bottom-pinned footer lines: separator, workspace
// path (muted NotAvailable placeholder when absent — never fabricated), then
// the version line last ("• kui <ver>" via buildinfo, omitted when absent).
func (m SidebarModel) footerLines(width int) []string {
	if width < 20 {
		width = 20
	}
	if m.width == 42 && width != 42 {
		if width > 42 {
			width = 42
		}
	}

	muted := m.styles.HomeMuted
	bodyStyle := lipgloss.NewStyle().
		Foreground(muted.GetForeground()).
		Faint(true).
		Width(width - 2)
	sep := muted.Render(strings.Repeat("─", width-2))

	lines := []string{sep}

	// Workspace path near the bottom (OpenCode rail layout)
	if m.workspace != "" {
		ws := m.workspace
		if lipgloss.Width(ws) > width-2 {
			ws = truncateSidebarLine(ws, width-2)
		}
		lines = append(lines, bodyStyle.Render(ws))
	} else {
		// NotAvailable muted when absent (never fabricate)
		lines = append(lines, bodyStyle.Render(muted.Render("NotAvailable")))
	}

	// Version line bottom-most via buildinfo: • kui <ver> when present else omitted
	if ver := getVersion(); ver != "" {
		footer := fmt.Sprintf("• kui %s", ver)
		// success dot uses accent? Use muted with success color if available
		if m.styles.Theme != nil && m.styles.Theme.Success != "" {
			dot := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.Success)).Render("•")
			footer = dot + " kui " + ver
		}
		lines = append(lines, bodyStyle.Render(footer))
	}

	return lines
}

func truncateSidebarLine(s string, max int) string {
	if lipgloss.Width(s) <= max {
		return s
	}
	out := ""
	for _, r := range s {
		if lipgloss.Width(out+string(r)) > max-3 {
			break
		}
		out += string(r)
	}
	return out + "..."
}
