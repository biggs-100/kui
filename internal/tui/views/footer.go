package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// FooterModel renders the session status bar.
// It mirrors routes/session/footer.tsx: a space-between row with the working
// directory in textMuted on the left and the connection cluster on the right
// (• N LSP + ⊙ N MCP + △ N + /status when connected; Get started ↔ /connect
// welcome cycle otherwise). Counts come from real sync.data.* or are omitted
// as muted, never fabricated.
type FooterModel struct {
	styles     *theme.Styles
	dir        string
	width      int
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

// SetWidth sets the row width used for the space-between composition.
func (m *FooterModel) SetWidth(w int) {
	m.width = w
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

// ClearLSP clears LSP to nil (muted NotAvailable) but keeps connected.
func (m *FooterModel) ClearLSP() {
	m.hasLSP = false
	m.lspCount = 0
}

// SetMCP sets MCP count and marks as connected.
func (m *FooterModel) SetMCP(count int) {
	m.hasMCP = true
	m.mcpCount = count
	m.connected = true
}

// ClearMCP clears MCP to nil (muted).
func (m *FooterModel) ClearMCP() {
	m.hasMCP = false
	m.mcpCount = 0
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

// Render produces the footer string: a space-between row — working directory
// in textMuted on the left, status cluster on the right. When connected, the
// cluster shows • N LSP + ⊙ N MCP + △ N + /status with counts from sync.data
// or muted omits; when not connected (welcome), it cycles Get started ↔
// /connect via tick.
func (m FooterModel) Render() string {
	if m.styles == nil {
		return ""
	}
	var right string
	if !m.connected {
		// Welcome tick cycles Get started → /connect
		if m.tick%2 == 0 {
			right = m.styles.HomeMuted.Render("Get started")
		} else {
			right = m.styles.HomeMuted.Render("/connect")
		}
	} else {
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
		right = strings.Join(parts, sep)
	}

	left := ""
	if m.dir != "" {
		left = m.styles.HomeMuted.Render(m.dir)
	}
	return joinSpaceBetween(left, right, m.width)
}

// joinSpaceBetween composes left and right on one row separated by enough
// spaces to fill width. Without a usable width it falls back to a two-space
// gap; when both segments cannot fit, the status cluster wins (the working
// directory remains visible in the rail footer).
func joinSpaceBetween(left, right string, width int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if width <= rw+1 {
		return right
	}
	if lw+rw+2 > width {
		return strings.Repeat(" ", max(0, width-rw)) + right
	}
	return left + strings.Repeat(" ", width-lw-rw) + right
}
