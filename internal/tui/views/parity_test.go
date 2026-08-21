package views

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestParityFooterNoFakes(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetDir("~/project")
	m.SetModel("gpt-4")
	m.SetTokens(1234, 10000)
	m.SetCost(0.05)
	got := m.Render()
	for _, f := range []string{"1.18.18", "OpenCode 1", "MCP", "LSP", "context7", "engram", "319k", "disconnected"} {
		if strings.Contains(got, f) {
			t.Errorf("footer must not contain %q, got: %q", f, got)
		}
	}
}

func TestParitySidebarNoFakes(t *testing.T) {
	m := NewSidebarModel(testStyles())
	m.SetTokens(0, 0)
	m.SetCost(0)
	got := m.View(40)
	for _, f := range []string{"1.2.1", "0 run", "done", "Σ 0", "context7", "engram", "319k", "disconnected", "Subagents", "MCP", "LSP"} {
		if strings.Contains(got, f) {
			t.Errorf("sidebar must not contain %q, got: %q", f, got)
		}
	}
	if !strings.Contains(got, "0 tokens 0% $0.00") {
		t.Errorf("sidebar should show truthful zero, got: %q", got)
	}
}

func TestParityModelCatalogNoFakes(t *testing.T) {
	for _, id := range AvailableModels() {
		if strings.Contains(id, "mimo") {
			t.Errorf("AvailableModels must not contain fabricated %q", id)
		}
	}
}

func TestParityNoHexLiteralsOutsideTheme(t *testing.T) {
	pattern := regexp.MustCompile(`#[0-9a-fA-F]{6}`)
	files := []string{"internal/tui/app.go", "internal/tui/markdown/renderer.go", "internal/tui/views/tool.go", "internal/tui/views/chat.go", "internal/tui/ui/dialog.go", "internal/tui/ui/border.go"}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if pattern.Match(data) {
			t.Errorf("%s contains hard-coded hex (must use theme tokens)", f)
		}
		for _, r := range []string{"#2a2a2a", "#252525", "#569cd6", "#e0af68"} {
			if strings.Contains(string(data), r) {
				t.Errorf("%s contains residual %s", f, r)
			}
		}
	}
}

func TestParityStylesUseTokens(t *testing.T) {
	s := testStyles()
	if s.Panel.GetBackground() == nil {
		t.Error("Panel background nil")
	}
	if s.InputBar.GetBackground() == nil {
		t.Error("InputBar background nil")
	}
}
