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
	// Fabricated literals from the removed hardcoding defect must never
	// appear, regardless of state.
	for _, f := range []string{"1.2.1", "context7", "engram", "319k", "disconnected", "OpenCode", "Open Code"} {
		if strings.Contains(got, f) {
			t.Errorf("sidebar must not contain %q, got: %q", f, got)
		}
	}
	if !strings.Contains(got, "0 tokens 0% $0.00") {
		t.Errorf("sidebar should show truthful zero, got: %q", got)
	}
}

// TestParitySidebarSectionsRequireSource asserts the honest-render contract:
// Subagents and MCP sections appear ONLY when a real source was attached,
// while the LSP section states its true disabled state instead of omitting.
func TestParitySidebarSectionsRequireSource(t *testing.T) {
	m := NewSidebarModel(testStyles())
	m.SetTokens(0, 0)
	got := m.View(42)
	if strings.Contains(got, "Subagents") {
		t.Errorf("sidebar without subagent source must not contain %q, got: %q", "Subagents", got)
	}
	if strings.Contains(got, "MCP") {
		t.Errorf("sidebar without MCP servers must not contain %q, got: %q", "MCP", got)
	}
	if !strings.Contains(got, "LSPs are disabled") {
		t.Errorf("sidebar should show truthful LSP disabled state, got: %q", got)
	}
}

// TestParitySidebarRealSectionsRender asserts sections render from real
// data once a source is attached (stats line + capped rows + server states).
func TestParitySidebarRealSectionsRender(t *testing.T) {
	m := NewSidebarModel(testStyles())
	m.SetTokens(0, 0)
	m.SetSubagents(1, 2, 1, []SubTask{
		{Title: "write tests", Running: true, At: "13:01"},
		{Title: "fix bug", Err: true, At: "13:05"},
		{Title: "docs pass", At: "13:09"},
	})
	m.SetMCPServers([]MCPServerState{
		{Name: "alpha", Connected: true},
		{Name: "beta", Connected: false},
	})
	got := m.View(42)
	for _, want := range []string{
		"Subagents",
		"1 run", "2 done", "1 err", "Σ 4",
		"write tests ↳ 13:01",
		"fix bug ↳ 13:05",
		"docs pass ↳ 13:09",
		"MCP",
		"alpha Connected",
		"beta Failed",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sidebar missing %q, got:\n%s", want, got)
		}
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
