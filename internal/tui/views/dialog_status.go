package views

import (
	"fmt"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/ui"
	"github.com/biggs-100/kui/internal/tui/util"
	"github.com/charmbracelet/lipgloss"
)

// MCPStatus enumerates MCP server states.
type MCPStatus string

const (
	MCPConnected MCPStatus = "connected"
	MCPFailed    MCPStatus = "failed"
	MCPDisabled  MCPStatus = "disabled"
	MCPNeedsAuth MCPStatus = "needs_auth"
)

// LSPStatus similarly.
type LSPStatus string

const (
	LSPConnected LSPStatus = "connected"
	LSPFailed    LSPStatus = "failed"
	LSPDisabled  LSPStatus = "disabled"
	LSPNeedsAuth LSPStatus = "needs_auth"
)

// MCPServerInfo holds MCP server display data.
type MCPServerInfo struct {
	Name   string
	Status MCPStatus
	Error  string
}

// LSPServerInfo holds LSP server display data.
type LSPServerInfo struct {
	Name   string
	Status LSPStatus
	Error  string
}

// FormatterInfo holds formatter display.
type FormatterInfo struct {
	Name   string
	Source string // file:// or name@version
}

// PluginInfo holds plugin display.
type PluginInfo struct {
	Name    string
	Version string
	Path    string // file:// if path else name@version
}

// DialogStatusModel renders MCP/LSP counts with colored dots + error details,
// formatters/plugins. Nil→muted for missing stores.
type DialogStatusModel struct {
	mcpServers []MCPServerInfo
	lspServers []LSPServerInfo
	formatters []FormatterInfo
	plugins    []PluginInfo
	width      int
	height     int
	styles     *theme.Styles
	quitting   bool
}

// NewDialogStatusModel creates a status dialog.
func NewDialogStatusModel(width, height int) DialogStatusModel {
	return DialogStatusModel{
		width:  width,
		height: height,
		styles: theme.NewStyles(theme.OpenCode()),
	}
}

func (m *DialogStatusModel) SetStyles(s *theme.Styles) {
	if s != nil {
		m.styles = s
	}
}

func (m *DialogStatusModel) SetMCP(servers []MCPServerInfo)  { m.mcpServers = servers }
func (m *DialogStatusModel) SetLSP(servers []LSPServerInfo)  { m.lspServers = servers }
func (m *DialogStatusModel) SetFormatters(f []FormatterInfo) { m.formatters = f }
func (m *DialogStatusModel) SetPlugins(p []PluginInfo)       { m.plugins = p }
func (m *DialogStatusModel) SetSize(w, h int)                { m.width = w; m.height = h }

func dotColorForStatus(status string, t *theme.Theme) string {
	switch status {
	case "connected":
		if t.Success != "" {
			return t.Success
		}
		return t.StatusOK
	case "failed":
		if t.Error != "" {
			return t.Error
		}
		return t.StatusError
	case "disabled":
		return t.TextMuted
	case "needs_auth":
		if t.Warning != "" {
			return t.Warning
		}
		return t.StatusWarn
	default:
		return t.TextMuted
	}
}

// View renders status dialog with backdrop 60/88/116, centered, counts with colored dots.
func (m DialogStatusModel) View() string {
	if m.quitting {
		return ""
	}
	var b strings.Builder
	// Header
	title := "Status"
	if m.styles != nil && m.styles.Theme != nil {
		title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.styles.Theme.Primary)).Render(title)
	}
	b.WriteString(title)
	b.WriteString("\n\n")

	// MCP Servers section
	mcpTitle := fmt.Sprintf("MCP Servers (%d)", len(m.mcpServers))
	if m.styles != nil && m.styles.Theme != nil {
		mcpTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.TextMuted)).Bold(true).Render(mcpTitle)
	}
	b.WriteString(mcpTitle)
	b.WriteString("\n")
	if len(m.mcpServers) == 0 {
		muted := "NotAvailable"
		if m.styles != nil && m.styles.Theme != nil {
			muted = lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.TextMuted)).Faint(true).Render(muted)
		}
		b.WriteString("  " + muted + "\n")
	} else {
		for _, s := range m.mcpServers {
			dot := "•"
			if m.styles != nil && m.styles.Theme != nil {
				color := dotColorForStatus(string(s.Status), m.styles.Theme)
				dot = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("•")
			}
			line := fmt.Sprintf("  %s %s", dot, s.Name)
			b.WriteString(line)
			b.WriteString("\n")
			if s.Error != "" {
				errLine := util.TruncateMiddle(s.Error, 76)
				if m.styles != nil && m.styles.Theme != nil {
					errLine = lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.TextMuted)).Faint(true).Render("    " + errLine)
				} else {
					errLine = "    " + errLine
				}
				b.WriteString(errLine)
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")
	// LSP Servers section
	lspTitle := fmt.Sprintf("LSP Servers (%d)", len(m.lspServers))
	if m.styles != nil && m.styles.Theme != nil {
		lspTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.TextMuted)).Bold(true).Render(lspTitle)
	}
	b.WriteString(lspTitle)
	b.WriteString("\n")
	if len(m.lspServers) == 0 {
		muted := "NotAvailable"
		if m.styles != nil && m.styles.Theme != nil {
			muted = lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.TextMuted)).Faint(true).Render(muted)
		}
		b.WriteString("  " + muted + "\n")
	} else {
		for _, s := range m.lspServers {
			dot := "•"
			if m.styles != nil && m.styles.Theme != nil {
				color := dotColorForStatus(string(s.Status), m.styles.Theme)
				dot = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("•")
			}
			line := fmt.Sprintf("  %s %s", dot, s.Name)
			b.WriteString(line)
			b.WriteString("\n")
			if s.Error != "" {
				errLine := util.TruncateMiddle(s.Error, 76)
				if m.styles != nil && m.styles.Theme != nil {
					errLine = lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.TextMuted)).Faint(true).Render("    " + errLine)
				} else {
					errLine = "    " + errLine
				}
				b.WriteString(errLine)
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")
	// Formatters section
	if len(m.formatters) > 0 {
		fTitle := "Formatters"
		if m.styles != nil && m.styles.Theme != nil {
			fTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.TextMuted)).Bold(true).Render(fTitle)
		}
		b.WriteString(fTitle)
		b.WriteString("\n")
		for _, f := range m.formatters {
			src := f.Source
			if src == "" {
				src = f.Name
			}
			// file:// or name@version already
			src = util.TruncateMiddle(src, 76)
			b.WriteString("  " + src + "\n")
		}
		b.WriteString("\n")
	}
	if len(m.plugins) > 0 {
		pTitle := "Plugins"
		if m.styles != nil && m.styles.Theme != nil {
			pTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(m.styles.Theme.TextMuted)).Bold(true).Render(pTitle)
		}
		b.WriteString(pTitle)
		b.WriteString("\n")
		for _, p := range m.plugins {
			var src string
			if strings.HasPrefix(p.Path, "file://") {
				src = p.Path
			} else if p.Version != "" {
				src = p.Name + "@" + p.Version
			} else {
				src = p.Name
			}
			src = util.TruncateMiddle(src, 76)
			b.WriteString("  " + src + "\n")
		}
		b.WriteString("\n")
	}
	content := strings.TrimSuffix(b.String(), "\n")
	// Use Dialog primitive for backdrop and centering 60/88/116
	size := 88
	if m.width > 0 {
		if m.width <= 60 {
			size = 60
		} else if m.width <= 88 {
			size = 88
		} else {
			size = 116
		}
	} else {
		if m.width < 80 {
			size = 60
		} else if m.width > 130 {
			size = 116
		}
	}
	d := ui.NewDialog(size, content)
	return d.View(m.width, m.height)
}

func (m DialogStatusModel) Quitting() bool      { return m.quitting }
func (m *DialogStatusModel) SetQuitting(v bool) { m.quitting = v }
