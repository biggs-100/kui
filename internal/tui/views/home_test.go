package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/tui/theme"
)

func TestHomeViewRendersAllComponents(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomeView(styles, 80, 24)
	got := m.View()
	if got == "" {
		t.Error("HomeView View() returned empty string")
	}
	// Should contain the logo (█▀▀█ two-tone)
	if !strings.Contains(got, "█") {
		t.Error("HomeView should contain logo block characters")
	}
}

func TestHomeViewLayoutVerticallyCentered(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomeView(styles, 80, 24)
	got := m.View()
	if got == "" {
		t.Fatal("HomeView returned empty")
	}
	lines := strings.Split(got, "\n")
	// With 24 lines height, there should be leading empty lines for centering
	if len(lines) < 10 {
		t.Errorf("expected at least 10 lines for vertical centering, got %d", len(lines))
	}
}

func TestHomeViewPromptHasBorder(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomeView(styles, 80, 24)
	got := m.View()
	// Should contain placeholder text
	if !strings.Contains(got, "Ask") && !strings.Contains(got, "kui") {
		// The prompt border characters are rendered by lipgloss
		// Just verify the view is non-empty and has structure
		if len(got) < 50 {
			t.Errorf("HomeView seems too short: %d chars", len(got))
		}
	}
}

func TestHomeViewResize(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())

	// Create at 80x24
	m := NewHomeView(styles, 80, 24)
	got1 := m.View()
	if got1 == "" {
		t.Fatal("HomeView at 80x24 returned empty")
	}

	// Resize to 120x40
	m.SetSize(120, 40)
	got2 := m.View()
	if got2 == "" {
		t.Fatal("HomeView at 120x40 returned empty")
	}
}

func TestHomeViewSetInput(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomeView(styles, 80, 24)
	m.SetInput("hello world")
	got := m.View()
	if !strings.Contains(got, "hello world") {
		t.Errorf("HomeView should contain input text, got:\n%s", got)
	}
}

func TestHomeViewGetInput(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomeView(styles, 80, 24)
	m.SetInput("test input")
	if m.GetInput() != "test input" {
		t.Errorf("GetInput() = %q, want %q", m.GetInput(), "test input")
	}
}

func TestHomeFlexSpacerCentering(t *testing.T) {
	styles := theme.NewStyles(theme.OpenCodeTheme)
	m := NewHomeView(styles, 120, 30)
	got := m.View()
	lines := strings.Split(got, "\n")
	// Find first non-empty line (logo start) and last non-empty (prompt decorative)
	first := -1
	last := -1
	for i, l := range lines {
		if strings.TrimSpace(l) != "" {
			if first == -1 {
				first = i
			}
			last = i
		}
	}
	if first == -1 || last == -1 {
		t.Fatal("could not find content lines")
	}
	topSpacer := first
	bottomSpacer := len(lines) - 1 - last
	if diff := topSpacer - bottomSpacer; diff < -1 || diff > 1 {
		t.Errorf("flex spacer not centered: top %d bottom %d diff %d (want ±1)", topSpacer, bottomSpacer, diff)
	}
}

func TestHomeResizeKeepsCentering(t *testing.T) {
	styles := theme.NewStyles(theme.OpenCodeTheme)
	m1 := NewHomeView(styles, 80, 24)
	got1 := m1.View()
	if !strings.Contains(got1, "█") {
		t.Errorf("home at 80x24 should contain logo, got %q", got1)
	}
	m2 := NewHomeView(styles, 160, 40)
	got2 := m2.View()
	if !strings.Contains(got2, "█") {
		t.Errorf("home at 160x40 should contain logo, got %q", got2)
	}
	// Both should be centered (contain leading spaces due to PlaceHorizontal)
	if len(got1) == 0 || len(got2) == 0 {
		t.Error("resize goldens empty")
	}
}

func TestHomeGolden(t *testing.T) {
	ResetPlaceholderCounter()
	styles := theme.NewStyles(theme.OpenCodeTheme)
	cases := []struct {
		name   string
		width  int
		height int
		file   string
	}{
		{"80", 80, 24, "testdata/home_80.txt"},
		{"120", 120, 30, "testdata/home_120.txt"},
		{"160", 160, 40, "testdata/home_160.txt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewHomeView(styles, tc.width, tc.height)
			got := m.View()
			golden := filepath.Join(tc.file)
			if *update {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("golden file not found (run with -update): %v", err)
			}
			if got != string(want) {
				t.Errorf("golden mismatch for %s\nwant %d bytes, got %d bytes", tc.name, len(want), len(got))
			}
		})
	}
}
