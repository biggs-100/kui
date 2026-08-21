package views

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/util"
	"github.com/charmbracelet/lipgloss"
)

// SubTask is one background sub-agent row for the sidebar.
type SubTask struct {
	Title   string
	Running bool
	Err     bool
	At      string // formatted HH:MM of start (running) or finish (done/err)
}

// MCPServerState is the runtime connection state of one MCP server.
type MCPServerState struct {
	Name      string
	Connected bool
}

// SidebarModel renders the opencode-style right sidebar.
// Width MUST be 42 cols (REQ-TUI-APP-2). Uses locale FormatNumber, header
// title+sessionID+workspace, footer version via buildinfo. Section data is
// real-state-only: sources absent → sections omitted, never fabricated.
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
	subSet     bool
	subRun     int
	subDone    int
	subErr     int
	subTasks   []SubTask
	mcpServers []MCPServerState
	modified   []ModifiedFile
}

// ModifiedFile is one real working-tree change: path plus +/- line counts.
type ModifiedFile struct {
	Name   string
	Added  int
	Removed int
}

// SetModifiedFiles feeds the Files section from real git state. Empty slice
// → the section is omitted entirely.
func (m *SidebarModel) SetModifiedFiles(files []ModifiedFile) {
	m.modified = files
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

// SetSubagents sets real background sub-agent stats and rows. Call only when
// a live source exists — it marks the section as present.
func (m *SidebarModel) SetSubagents(run, done, errCount int, tasks []SubTask) {
	m.subSet = true
	m.subRun = run
	m.subDone = done
	m.subErr = errCount
	m.subTasks = tasks
}

// SetMCPServers sets real per-server MCP connection states. Empty slice
// omits the section.
func (m *SidebarModel) SetMCPServers(servers []MCPServerState) {
	m.mcpServers = servers
}

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
// bottom. Following the upstream design language, the rail is a PURE
// background panel: no border anywhere — elevation comes from the
// backgroundPanel fill being one step lighter than the terminal background.
// The block spans exactly width columns and height rows with padding 1 row
// top/bottom and 2 columns left/right (the left padding doubles as the
// gutter against the main column). When the content is taller than height
// the block renders at natural height.
func (m SidebarModel) ViewFullHeight(width, height int) string {
	if m.styles == nil || width < 10 {
		return ""
	}
	if height < 1 {
		return m.View(width)
	}
	body := m.viewBody(width - 2)
	footer := m.footerLines(width - 2)
	fill := height - 2 - len(strings.Split(body, "\n")) - len(footer)
	if fill < 0 {
		fill = 0
	}

	var b strings.Builder
	b.WriteString(body)
	for i := 0; i < fill; i++ {
		b.WriteString("\n")
	}
	for _, l := range footer {
		b.WriteString("\n")
		b.WriteString(l)
	}
	rail := m.styles.Sidebar.Copy().
		Width(width).
		Padding(1, 2)
	return rail.Render(b.String())
}

// railBg returns the rail's background color. Every style that renders
// inside the rail must carry it explicitly: an internal SGR reset kills the
// container background for the rest of the line, leaving terminal-default
// holes behind any segment that lacks its own bg.
func (m SidebarModel) railBg() lipgloss.Color {
	if m.styles.Theme != nil && m.styles.Theme.BackgroundPanel != "" {
		return lipgloss.Color(m.styles.Theme.BackgroundPanel)
	}
	return lipgloss.Color("")
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

	// Section headers follow the upstream language: plain bold words in the
	// text color (never accent, never decorated), bodies in textMuted,
	// sections separated by whitespace only — no rules, no glyphs.
	// EVERY style below carries the rail background explicitly: an internal
	// SGR reset kills the container background for the rest of the line, so
	// any segment without its own bg renders over terminal-default holes.
	panelBg := m.railBg()
	headerStyle := lipgloss.NewStyle().Bold(true).Background(panelBg)
	if m.styles.Theme != nil && m.styles.Theme.Text != "" {
		headerStyle = headerStyle.Foreground(lipgloss.Color(m.styles.Theme.Text))
	}

	muted := m.styles.HomeMuted
	// No inner .Width(): lipgloss emits a Width-style's padding OUTSIDE its
	// SGR run, which punches unpainted (terminal-default) holes through the
	// rail's background fill. Lines are length-capped by truncation instead;
	// every style carries the rail bg so text runs self-paint.
	bodyStyle := lipgloss.NewStyle().
		Foreground(muted.GetForeground()).
		Background(panelBg)

	var b strings.Builder

	firstSection := true
	section := func(title string, lines []string) {
		if !firstSection {
			b.WriteString("\n")
		}
		firstSection = false
		b.WriteString(headerStyle.Render(title))
		b.WriteString("\n")
		for _, l := range lines {
			if lipgloss.Width(l) > width-2 {
				l = truncateSidebarLine(l, width-2)
			}
			b.WriteString(bodyStyle.Render(l))
			b.WriteString("\n")
		}
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

	// Subagents section — real background task state only (source absent or
	// zero tasks → omitted). Matches OpenCode rail order: above Context.
	if m.subSet && m.subRun+m.subDone+m.subErr > 0 {
		success := ""
		errColor := ""
		warn := ""
		if m.styles.Theme != nil {
			success = m.styles.Theme.Success
			errColor = m.styles.Theme.Error
			warn = m.styles.Theme.Warning
		}
		glyph := func(s SubTask) string {
			switch {
			case s.Running:
				return lipgloss.NewStyle().Foreground(lipgloss.Color(warn)).Background(panelBg).Render("●")
			case s.Err:
				return lipgloss.NewStyle().Foreground(lipgloss.Color(errColor)).Background(panelBg).Render("×")
			default:
				return lipgloss.NewStyle().Foreground(lipgloss.Color(success)).Background(panelBg).Render("✓")
			}
		}
		stats := fmt.Sprintf("%s %d run · %s %d done · %s %d err · Σ %d",
			lipgloss.NewStyle().Foreground(lipgloss.Color(warn)).Background(panelBg).Render("●"),
			m.subRun,
			lipgloss.NewStyle().Foreground(lipgloss.Color(success)).Background(panelBg).Render("✓"),
			m.subDone,
			lipgloss.NewStyle().Foreground(lipgloss.Color(errColor)).Background(panelBg).Render("×"),
			m.subErr,
			m.subRun+m.subDone+m.subErr,
		)
		var subLines []string
		subLines = append(subLines, stats)
		for _, t := range m.subTasks {
			row := fmt.Sprintf("%s %s ↳ %s", glyph(t), t.Title, t.At)
			subLines = append(subLines, row)
		}
		section("Subagents", subLines)
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

	// MCP section — real per-server connection states only (none configured
	// or attempted → omitted; never a fabricated zero-state).
	if len(m.mcpServers) > 0 {
		errColor := ""
		success := ""
		if m.styles.Theme != nil {
			errColor = m.styles.Theme.Error
			success = m.styles.Theme.Success
		}
		var mcpLines []string
		for _, s := range m.mcpServers {
			if s.Connected {
				mcpLines = append(mcpLines, fmt.Sprintf("%s %s",
					s.Name,
					lipgloss.NewStyle().Foreground(lipgloss.Color(success)).Background(panelBg).Render("Connected"),
				))
			} else {
				mcpLines = append(mcpLines, fmt.Sprintf("%s %s",
					s.Name,
					lipgloss.NewStyle().Foreground(lipgloss.Color(errColor)).Background(panelBg).Render("Failed"),
				))
			}
		}
		section("MCP", mcpLines)
	}

	// LSP section — kui's TUI has no LSP server management wired today, so
	// the honest state is "disabled" (mirrors OpenCode's disabled rail line).
	// Update when real LSP tracking lands; do not fabricate server rows.
	section("LSP", []string{"LSPs are disabled"})

	// Modified Files — real working-tree changes (git), refreshed by the app
	// on its periodic tick. Empty → omitted, never fabricated.
	if len(m.modified) > 0 {
		addStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.styles.Theme.DiffAdded)).
			Background(panelBg)
		remStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.styles.Theme.DiffRemoved)).
			Background(panelBg)
		nameFg := m.styles.Theme.TextMuted
		var fileLines []string
		for _, f := range m.modified {
			name := f.Name
			if lipgloss.Width(name) > width-12 {
				name = truncateSidebarLine(name, width-12)
			}
			row := lipgloss.NewStyle().
				Foreground(lipgloss.Color(nameFg)).
				Background(panelBg).Render(name) + " " +
				addStyle.Render(fmt.Sprintf("+%d", f.Added)) + " " +
				remStyle.Render(fmt.Sprintf("-%d", f.Removed))
			fileLines = append(fileLines, row)
		}
		section("Files", fileLines)
	}

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
		Background(m.railBg())

	lines := []string{}

	// Workspace path near the bottom (OpenCode rail layout)
	if m.workspace != "" {
		ws := m.workspace
		if lipgloss.Width(ws) > width-2 {
			ws = truncateSidebarLine(ws, width-2)
		}
		lines = append(lines, bodyStyle.Render(ws))
	} else {
		// NotAvailable muted when absent (never fabricate)
		lines = append(lines, bodyStyle.Render("NotAvailable"))
	}

	// Version line bottom-most via buildinfo: • kui <ver> when present else omitted
	if ver := getVersion(); ver != "" {
		footer := fmt.Sprintf("• kui %s", ver)
		// success dot uses accent? Use muted with success color if available
		if m.styles.Theme != nil && m.styles.Theme.Success != "" {
			dot := lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.styles.Theme.Success)).
				Background(m.railBg()).Render("•")
			name := lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color(m.styles.Theme.Text)).
				Background(m.railBg()).Render("kui")
			verText := bodyStyle.Render(ver)
			footer = dot + " " + name + " " + verText
		}
		// Cap the visible width so a long buildinfo never wraps inside the
		// rail (wrapping would spill onto a second unpainted row).
		if lipgloss.Width(footer) > width-2 {
			footer = truncateSidebarLine(footer, width-2)
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
