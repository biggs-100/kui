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

// SetWorkspace sets workspace path displayed in header.
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

	// Header: title+sessionID+workspace (REQ-TUI-APP-2, REQ-TUI-CHAT-6)
	var headerLines []string
	if m.title != "" {
		headerLines = append(headerLines, m.title)
	}
	if m.sessionID != "" {
		headerLines = append(headerLines, "session "+m.sessionID)
	}
	if m.workspace != "" {
		headerLines = append(headerLines, m.workspace)
	} else {
		// NotAvailable muted when absent (never fabricate)
		headerLines = append(headerLines, muted.Render("NotAvailable"))
	}
	if len(headerLines) > 0 {
		section("Workspace", headerLines)
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

	// Footer version via buildinfo: • Open Code <ver> when present else omitted
	if ver := getVersion(); ver != "" {
		footer := fmt.Sprintf("• Open Code %s", ver)
		// success dot uses accent? Use muted with success color if available
		if m.styles.Theme != nil && m.styles.Theme.Success != "" {
			dot := lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.Success)).Render("•")
			footer = dot + " Open Code " + ver
		}
		b.WriteString(bodyStyle.Render(footer))
		b.WriteString("\n")
		b.WriteString(sep)
		b.WriteString("\n")
	}

	content := strings.TrimSuffix(b.String(), "\n")
	// Use the theme's sidebar style (background BGSidebar + padding) instead of
	// a hardcoded hex literal.
	return m.styles.Sidebar.Width(width).Render(content)
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
