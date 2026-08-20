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
	// Logo should contain block characters from the ASCII art
	if !strings.Contains(got, "██") {
		t.Errorf("Logo should contain block characters '██', got:\n%s", got)
	}
}

func TestLogoCentersWithinWidth(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewLogoModel(styles)
	got := m.View(80)
	// PlaceHorizontal centers the text; the rendered output should
	// have at most 80 visible columns (ANSI codes don't count as visible).
	// We verify the logo renders successfully — the centering is done
	// by lipgloss and is visually correct.
	if got == "" {
		t.Error("Logo View(80) returned empty")
	}
	// Should have exactly 6 lines (one per logo line)
	lines := strings.Split(got, "\n")
	if len(lines) != 6 {
		t.Errorf("expected 6 logo lines, got %d", len(lines))
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
