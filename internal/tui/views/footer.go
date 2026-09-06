package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// FooterModel renders the pi-style 2-line dim footer (REQ-TUI-APP-6,
// REQ-TUI-PROF-1/4): L1 `{cwd} ({branch}) {session}`; L2 left stats, right
// `(provider) model` + thinking. Unknown branch/tokens/session are omitted,
// never fabricated. There are no profile tabs; the active profile surfaces
// on L2 right, with a TAB hint when several profiles exist.
type FooterModel struct {
	styles *theme.Styles
	width  int

	dir     string
	branch  string
	session string

	tokens     int
	contextMax int
	cost       float64
	hasCost    bool

	provider string
	model    string
	thinking string

	profiles []string
	active   int

	// Legacy sync fields retained for API compat (MCP/LSP now live in
	// /status per design). Ignored by Render.
	connected bool
	hasLSP    bool
	lspCount  int
	hasMCP    bool
	mcpCount  int
	hasPerm   bool
	permCount int
	tick      int
}

// NewFooterModel creates a FooterModel.
func NewFooterModel(styles *theme.Styles) FooterModel {
	return FooterModel{styles: styles}
}

// SetWidth sets the row width used for truncation and space-between.
func (m *FooterModel) SetWidth(w int) {
	m.width = w
}

// SetDir sets the working directory (real cwd, home-shortened by caller).
func (m *FooterModel) SetDir(dir string) {
	m.dir = dir
}

// SetBranch sets the git branch; empty omits it (never fabricated).
func (m *FooterModel) SetBranch(branch string) {
	m.branch = branch
}

// SetSession sets the session name/ID shown on L1; empty omits it.
func (m *FooterModel) SetSession(session string) {
	m.session = session
}

// SetModel sets the current model name shown on L2 right.
func (m *FooterModel) SetModel(model string) {
	m.model = model
}

// SetProvider sets the provider name shown on L2 right as (provider).
func (m *FooterModel) SetProvider(provider string) {
	m.provider = provider
}

// SetThinking sets the thinking indicator shown on L2 right; empty omits it.
func (m *FooterModel) SetThinking(thinking string) {
	m.thinking = thinking
}

// SetProfiles records the profile list for the TAB hint; the active profile
// itself surfaces via provider/model on L2 right.
func (m *FooterModel) SetProfiles(profiles []string, active int) {
	m.profiles = profiles
	m.active = active
}

// SetTokens sets token count and context limit. Zero values are omitted,
// never rendered as fabricated "0".
func (m *FooterModel) SetTokens(total, limit int) {
	m.tokens = total
	m.contextMax = limit
}

// SetCost sets the session cost; shown only when explicitly set.
func (m *FooterModel) SetCost(cost float64) {
	m.cost = cost
	m.hasCost = true
}

// SetConnected marks the session connected (compat; ignored by Render).
func (m *FooterModel) SetConnected(connected bool) {
	m.connected = connected
}

// SetLSP records LSP count (compat; MCP/LSP live in /status now).
func (m *FooterModel) SetLSP(count int) {
	m.hasLSP = true
	m.lspCount = count
	m.connected = true
}

// ClearLSP clears LSP (compat).
func (m *FooterModel) ClearLSP() {
	m.hasLSP = false
	m.lspCount = 0
}

// SetMCP records MCP count (compat; MCP/LSP live in /status now).
func (m *FooterModel) SetMCP(count int) {
	m.hasMCP = true
	m.mcpCount = count
	m.connected = true
}

// ClearMCP clears MCP (compat).
func (m *FooterModel) ClearMCP() {
	m.hasMCP = false
	m.mcpCount = 0
}

// SetPerm sets permission count (compat; ignored by Render).
func (m *FooterModel) SetPerm(count int) {
	m.hasPerm = true
	m.permCount = count
}

// Tick advances the legacy welcome cycle (compat; ignored by Render).
func (m *FooterModel) Tick() {
	m.tick++
}

// line1 composes `{cwd} ({branch}) {session}`, omitting unknowns.
func (m FooterModel) line1() string {
	var segs []string
	if m.dir != "" {
		segs = append(segs, m.dir)
	}
	if m.branch != "" {
		segs = append(segs, "("+m.branch+")")
	}
	if m.session != "" {
		segs = append(segs, m.session)
	}
	return strings.Join(segs, " ")
}

// statsLeft composes the L2-left stats from real data only. With no
// discoverable profiles it renders the muted "no profiles available" hint
// (REQ-TUI-PROF-4); the whole footer line is already dim via HomeMuted.
func (m FooterModel) statsLeft() string {
	var segs []string
	if m.tokens > 0 || m.contextMax > 0 {
		if m.contextMax > 0 && m.tokens > 0 {
			pct := m.tokens * 100 / m.contextMax
			segs = append(segs, fmt.Sprintf("%s tokens %d%%", formatInt(m.tokens), pct))
		} else if m.tokens > 0 {
			segs = append(segs, fmt.Sprintf("%s tokens", formatInt(m.tokens)))
		}
	}
	if m.hasCost {
		segs = append(segs, fmt.Sprintf("$%.2f", m.cost))
	}
	if len(m.profiles) == 0 {
		segs = append(segs, "no profiles available")
	} else if len(m.profiles) > 1 {
		segs = append(segs, "TAB ⇆")
	}
	return strings.Join(segs, " · ")
}

// identityRight composes the L2-right `(provider) model` + thinking,
// omitting unknowns.
func (m FooterModel) identityRight() string {
	var b strings.Builder
	if m.provider != "" {
		b.WriteString("(" + m.provider + ")")
	}
	if m.model != "" {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(m.model)
	}
	if m.thinking != "" {
		if b.Len() > 0 {
			b.WriteString(" · ")
		}
		b.WriteString(m.thinking)
	}
	return b.String()
}

// Render produces exactly 2 dim lines (REQ-TUI-APP-6). Unknowns are omitted;
// nothing is fabricated. Lines truncate to width, never overflow.
func (m FooterModel) Render() string {
	if m.styles == nil {
		return "\n"
	}
	l1 := truncateFooter(m.line1(), m.width)
	left := truncateFooter(m.statsLeft(), m.width)
	right := truncateFooter(m.identityRight(), m.width)
	l2 := joinSpaceBetween(left, right, m.width)
	return m.styles.HomeMuted.Render(l1) + "\n" + m.styles.HomeMuted.Render(l2)
}

// joinSpaceBetween composes left and right on one row separated by enough
// spaces to fill width. Without a usable width it falls back to a two-space
// gap; when both segments cannot fit, the identity cluster wins.
func joinSpaceBetween(left, right string, width int) string {
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if width <= rw+1 {
		return right
	}
	if lw+rw+1 > width {
		return strings.Repeat(" ", max(0, width-rw)) + right
	}
	return left + strings.Repeat(" ", width-lw-rw) + right
}

// truncateFooter shortens s to width visible columns.
func truncateFooter(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	out := ""
	for _, r := range s {
		if lipgloss.Width(out+string(r)) > width {
			break
		}
		out += string(r)
	}
	return out
}

// formatInt renders n with thousands separators (locale-style, honest).
func formatInt(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := ""
	if strings.HasPrefix(s, "-") {
		neg, s = "-", s[1:]
	}
	var out []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return neg + string(out)
}
