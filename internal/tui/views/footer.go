package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
)

// FooterModel renders the session status bar.
// It mirrors routes/session/footer.tsx: when connected shows • N LSP + ⊙ N MCP + △ N + /status;
// when welcome (not connected) cycles Get started /connect via tick.
// Counts come from real sync.data.* or are omitted as muted, never fabricated.
type FooterModel struct {
	styles     *theme.Styles
	dir        string
	model      string
	tokens     int
	contextMax int
	cost       float64

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

// SetDir sets the working directory (retained for compatibility; not fabricated in session footer).
func (m *FooterModel) SetDir(dir string) {
	m.dir = dir
}

// SetModel sets the current model name (retained for compatibility).
func (m *FooterModel) SetModel(model string) {
	m.model = model
}

// SetTokens sets token count and context limit (retained for compatibility).
func (m *FooterModel) SetTokens(total, limit int) {
	m.tokens = total
	m.contextMax = limit
}

// SetCost sets the session cost (retained for compatibility).
func (m *FooterModel) SetCost(cost float64) {
	m.cost = cost
}

// SetConnected sets whether the session is connected (sync.data present).
func (m *FooterModel) SetConnected(connected bool) {
	m.connected = connected
}

// SetLSP sets LSP count and marks as connected (real sync.data).
func (m *FooterModel) SetLSP(count int) {
	m.hasLSP = true
	m.lspCount = count
	m.connected = true
}

// SetMCP sets MCP count and marks as connected.
func (m *FooterModel) SetMCP(count int) {
	m.hasMCP = true
	m.mcpCount = count
	m.connected = true
}

// SetPerm sets permission count (△).
func (m *FooterModel) SetPerm(count int) {
	m.hasPerm = true
	m.permCount = count
}

// Tick advances the welcome cycle (10s tick fires).
func (m *FooterModel) Tick() {
	m.tick++
}

// Render produces the footer string.
// When connected, shows • N LSP + ⊙ N MCP + △ N + /status with counts from sync.data or muted omit.
// When not connected (welcome), cycles Get started ↔ /connect via tick.
func (m FooterModel) Render() string {
	if m.styles == nil {
		return ""
	}
	if !m.connected {
		// Welcome tick cycles Get started → /connect
		if m.tick%2 == 0 {
			return m.styles.HomeMuted.Render("Get started")
		}
		return m.styles.HomeMuted.Render("/connect")
	}
	var parts []string
	// LSP: • N when hasLSP, otherwise muted omit (not 0 faked)
	if m.hasLSP {
		parts = append(parts, fmt.Sprintf("• %d LSP", m.lspCount))
	} else {
		parts = append(parts, m.styles.HomeMuted.Render("— LSP"))
	}
	if m.hasMCP {
		parts = append(parts, fmt.Sprintf("⊙ %d MCP", m.mcpCount))
	} else {
		parts = append(parts, m.styles.HomeMuted.Render("— MCP"))
	}
	if m.hasPerm {
		parts = append(parts, fmt.Sprintf("△ %d", m.permCount))
	}
	parts = append(parts, m.styles.HomeMuted.Render("/status"))
	sep := m.styles.HomeMuted.Render(" • ")
	return strings.Join(parts, sep)
}
