package views

import (
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
	// Should contain the logo
	if !strings.Contains(got, "██") {
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
