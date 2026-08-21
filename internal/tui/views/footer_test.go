package views

import (
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/tui/theme"
)

func TestNewFooterModel(t *testing.T) {
	m := NewFooterModel(testStyles())
	got := m.Render()
	if got == "" {
		t.Error("NewFooterModel should produce non-empty render (welcome tick)")
	}
	// Welcome mode should show Get started or /connect, not fabricated tokens
	if !strings.Contains(got, "Get started") && !strings.Contains(got, "/connect") {
		t.Errorf("welcome footer should contain 'Get started' or '/connect', got: %q", got)
	}
	if strings.Contains(got, "tokens") || strings.Contains(got, "$") {
		t.Errorf("session footer should not show fabricated tokens/cost in welcome, got: %q", got)
	}
}

func TestFooterConnectedShowsDots(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetLSP(2)
	m.SetMCP(1)
	// SetConnected is implied by SetLSP/SetMCP
	got := m.Render()
	if !strings.Contains(got, "• 2") {
		t.Errorf("connected footer should contain '• 2', got: %q", got)
	}
	if !strings.Contains(got, "⊙ 1") {
		t.Errorf("connected footer should contain '⊙ 1', got: %q", got)
	}
	if !strings.Contains(got, "/status") {
		t.Errorf("connected footer should contain '/status', got: %q", got)
	}
}

func TestFooterWelcomeTickCycles(t *testing.T) {
	m := NewFooterModel(testStyles())
	// Initially tick 0 -> Get started
	got1 := m.Render()
	if !strings.Contains(got1, "Get started") {
		t.Errorf("initial welcome should be 'Get started', got: %q", got1)
	}
	m.Tick()
	got2 := m.Render()
	if !strings.Contains(got2, "/connect") {
		t.Errorf("after tick should be '/connect', got: %q", got2)
	}
	m.Tick()
	got3 := m.Render()
	if !strings.Contains(got3, "Get started") {
		t.Errorf("after second tick should cycle to 'Get started', got: %q", got3)
	}
}

func TestFooterNoFabricationWhenAbsent(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetConnected(true)
	// No LSP/MCP counts set → should omit as muted, not 0 faked as connected
	got := m.Render()
	if strings.Contains(got, "• 0") || strings.Contains(got, "⊙ 0") {
		t.Errorf("absent sync.data should not fake 0 counts, got: %q", got)
	}
	// Should show muted placeholder for absent
	if !strings.Contains(got, "— LSP") && !strings.Contains(got, "— MCP") && !strings.Contains(got, "/status") {
		t.Errorf("absent counts should be omitted as muted, got: %q", got)
	}
}

func TestFooterPermissionTriangle(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetLSP(1)
	m.SetMCP(1)
	m.SetPerm(3)
	got := m.Render()
	if !strings.Contains(got, "△ 3") {
		t.Errorf("connected footer with perm should contain '△ 3', got: %q", got)
	}
}

func TestFooterTheme(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewFooterModel(styles)
	m.SetConnected(true)
	m.SetLSP(1)
	got := m.Render()
	if got == "" {
		t.Error("themed footer should render non-empty")
	}
}
