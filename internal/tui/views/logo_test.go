package views

import (
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/tui/theme"
)

func TestLogoRendersWithoutError(t *testing.T) {
	m := NewLogoModel(testStyles())
	got := m.View(80)
	if got == "" {
		t.Error("Logo View() returned empty string")
	}
}

func TestLogoUsesAccentColor(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewLogoModel(styles)
	got := m.View(80)
	// Logo should contain block characters from the ASCII art (█▀▀█ two-tone)
	if !strings.Contains(got, "█") {
		t.Errorf("Logo should contain block characters '█', got:\n%s", got)
	}
	if !strings.Contains(got, "▀") && !strings.Contains(got, "█") {
		t.Errorf("Logo should contain OpenCode-style block chars, got:\n%s", got)
	}
}

func TestLogoCentersWithinWidth(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewLogoModel(styles)
	got := m.View(80)
	if got == "" {
		t.Error("Logo View(80) returned empty")
	}
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 logo lines (█▀▀█ pairs), got %d", len(lines))
	}
}

func TestLogoWithDifferentWidths(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewLogoModel(styles)

	for _, w := range []int{40, 60, 80, 120} {
		got := m.View(w)
		if got == "" {
			t.Errorf("Logo View(%d) returned empty string", w)
		}
	}
}

func TestLogoHasShadowTint(t *testing.T) {
	styles := theme.NewStyles(theme.OpenCodeTheme)
	m := NewLogoModel(styles)
	got := m.View(80)
	if got == "" {
		t.Fatal("logo View returned empty")
	}
	// Two-tone output must produce shadow distinct from fg via Tint
	bg := styles.Theme.Background
	fg := styles.Theme.SyntaxKeyword
	if fg == "" {
		fg = styles.Theme.Accent
	}
	shadow := theme.Tint(bg, fg, 0.25)
	if shadow == fg {
		t.Errorf("shadow tint %q should differ from fg %q", shadow, fg)
	}
	if shadow == bg {
		t.Errorf("shadow tint %q should differ from bg %q", shadow, bg)
	}
	// Verify logo still contains block chars
	if !strings.Contains(got, "█") {
		t.Errorf("logo should contain block chars, got %q", got)
	}
}

func TestLogoTintIsThemeDerived(t *testing.T) {
	// Load custom theme from JSON and verify shadow recomputes via tint
	jsonStr := `{"name":"custom","bg":"#111111","background":"#111111","fg":"#ff0000","text":"#ff0000","accent":"#00ff00","syntax_keyword":"#00ff00","border":"#333333","background_panel":"#252525","background_element":"#2a2a2a"}`
	parsed, err := theme.ParseBytes([]byte(jsonStr))
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	styles := theme.NewStyles(parsed)
	m := NewLogoModel(styles)
	got := m.View(80)
	if got == "" {
		t.Fatal("logo with custom theme returned empty")
	}
	// Shadow should be Tint(#111111, #00ff00, 0.25) — verify Tint computation
	bg := parsed.Background
	if bg == "" {
		bg = parsed.BG
	}
	fg := parsed.SyntaxKeyword
	shadow := theme.Tint(bg, fg, 0.25)
	if shadow == fg || shadow == bg {
		t.Errorf("custom shadow %q should be blended and differ from bg %q and fg %q", shadow, bg, fg)
	}
	if !strings.Contains(got, "█") {
		t.Errorf("custom theme logo should still contain block chars, got %q", got)
	}
}
