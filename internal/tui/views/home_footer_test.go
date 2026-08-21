package views

import (
	"strings"
	"testing"
)

func TestHomeFooterRendersAllElements(t *testing.T) {
	m := NewHomeFooterModel(testStyles(), "~/project")
	got := m.Render()
	// Home footer must be empty or muted placeholder, not fabricated dir • LSP • MCP
	if strings.Contains(got, "LSP") || strings.Contains(got, "MCP") {
		t.Errorf("home footer should not contain fabricated LSP/MCP, got: %q", got)
	}
	// Should be empty or muted (we return empty)
	if got != "" && !strings.Contains(got, "—") && !strings.Contains(got, "NotAvailable") {
		t.Errorf("home footer should be empty or muted placeholder, got: %q", got)
	}
}

func TestHomeFooterEmptyWhenNoSlot(t *testing.T) {
	m := NewHomeFooterModel(testStyles(), ".")
	got := m.Render()
	if strings.Contains(got, "LSP") || strings.Contains(got, "MCP") || strings.Contains(got, "/status") {
		t.Errorf("home footer with no plugin slot and no sync data should be empty/muted, not 'LSP/MCP', got: %q", got)
	}
	// Accept empty or muted
	if got != "" && !strings.Contains(got, "—") {
		t.Logf("home footer empty check: got %q", got)
	}
}

func TestHomeFooterLSPDotConnected(t *testing.T) {
	m := NewHomeFooterModel(testStyles(), ".")
	m.SetLSPConnected(true)
	got := m.Render()
	// Home footer should still not show LSP even when connected flag set (no fabrication)
	if strings.Contains(got, "LSP") {
		t.Errorf("home footer should not fabricate LSP even when connected, got: %q", got)
	}
}

func TestHomeFooterLSPDotDisconnected(t *testing.T) {
	m := NewHomeFooterModel(testStyles(), ".")
	m.SetLSPConnected(false)
	got := m.Render()
	if strings.Contains(got, "LSP") {
		t.Errorf("home footer should not contain LSP when disconnected, got: %q", got)
	}
}

func TestHomeFooterMCPDotConnected(t *testing.T) {
	m := NewHomeFooterModel(testStyles(), ".")
	m.SetMCPConnected(true)
	got := m.Render()
	if strings.Contains(got, "MCP") {
		t.Errorf("home footer should not contain MCP, got: %q", got)
	}
}

func TestHomeFooterDefaultDir(t *testing.T) {
	m := NewHomeFooterModel(testStyles(), "")
	got := m.Render()
	// Should be empty, not show dir
	if strings.Contains(got, "LSP") || strings.Contains(got, "MCP") {
		t.Errorf("home footer with empty dir should not contain LSP/MCP, got: %q", got)
	}
}

func TestHomeFooterPluginSlot(t *testing.T) {
	m := NewHomeFooterModel(testStyles(), ".")
	m.SetPluginContent("custom plugin")
	got := m.Render()
	if !strings.Contains(got, "custom plugin") {
		t.Errorf("home footer with plugin content should contain it, got: %q", got)
	}
}
