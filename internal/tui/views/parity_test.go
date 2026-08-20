package views

import (
	"strings"
	"testing"
)

// RED: footer must not render fabricated version / status literals.
func TestParityFooterNoFakes(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetDir("~/project")
	m.SetModel("gpt-4")
	m.SetTokens(1234, 10000)
	m.SetCost(0.05)

	got := m.Render()

	forbidden := []string{
		"1.18.18", "OpenCode 1", "MCP", "LSP",
		"context7", "engram", "319k", "disconnected",
	}
	for _, f := range forbidden {
		if strings.Contains(got, f) {
			t.Errorf("footer must not contain fabricated %q, got: %q", f, got)
		}
	}
}

// RED: sidebar must not render fabricated version / subagent / MCP / LSP literals.
func TestParitySidebarNoFakes(t *testing.T) {
	m := NewSidebarModel(testStyles())
	m.SetTokens(0, 0)
	m.SetCost(0)

	got := m.View(40)

	forbidden := []string{
		"1.2.1", "0 run", "done", "Σ 0",
		"context7", "engram", "319k", "disconnected",
		"Subagents", "MCP", "LSP",
	}
	for _, f := range forbidden {
		if strings.Contains(got, f) {
			t.Errorf("sidebar must not contain fabricated %q, got: %q", f, got)
		}
	}
	// Truthful zero state when no data.
	if !strings.Contains(got, "0 tokens 0% $0.00") {
		t.Errorf("sidebar should show truthful zero, got: %q", got)
	}
}

// RED: model catalog must not contain fabricated IDs.
func TestParityModelCatalogNoFakes(t *testing.T) {
	for _, id := range AvailableModels() {
		if strings.Contains(id, "mimo") {
			t.Errorf("AvailableModels must not contain fabricated %q", id)
		}
	}
}
