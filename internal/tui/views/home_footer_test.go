package views

import (
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/tui/theme"
)

func TestHomeFooterRendersAllElements(t *testing.T) {
	m := NewHomeFooterModel(testStyles(), "~/project")
	got := m.Render()
	if got == "" {
		t.Error("HomeFooterModel Render() returned empty string")
	}
	// Should contain directory
	if !strings.Contains(got, "~/project") {
		t.Errorf("footer should contain directory, got: %q", got)
	}
	// Should contain LSP indicator
	if !strings.Contains(got, "LSP") {
		t.Errorf("footer should contain LSP, got: %q", got)
	}
	// Should contain MCP indicator
	if !strings.Contains(got, "MCP") {
		t.Errorf("footer should contain MCP, got: %q", got)
	}
	// Should contain /status
	if !strings.Contains(got, "/status") {
		t.Errorf("footer should contain /status, got: %q", got)
	}
}

func TestHomeFooterLSPDotConnected(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomeFooterModel(styles, ".")
	m.SetLSPConnected(true)
	got := m.Render()
	// Green dot = connected — check for the bullet character
	if !strings.Contains(got, "●") && !strings.Contains(got, "•") {
		t.Errorf("connected LSP should show dot indicator, got: %q", got)
	}
}

func TestHomeFooterLSPDotDisconnected(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomeFooterModel(styles, ".")
	m.SetLSPConnected(false)
	got := m.Render()
	// Should still render without error
	if got == "" {
		t.Error("disconnected footer should still render")
	}
}

func TestHomeFooterMCPDotConnected(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomeFooterModel(styles, ".")
	m.SetMCPConnected(true)
	got := m.Render()
	if !strings.Contains(got, "MCP") {
		t.Errorf("footer should contain MCP, got: %q", got)
	}
}

func TestHomeFooterDefaultDir(t *testing.T) {
	m := NewHomeFooterModel(testStyles(), "")
	got := m.Render()
	if got == "" {
		t.Error("footer with empty dir should still render")
	}
}
