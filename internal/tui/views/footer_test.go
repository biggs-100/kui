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
		t.Error("NewFooterModel should produce non-empty render")
	}
	// Default render should show placeholder dashes for empty fields
	if !strings.Contains(got, "—") && !strings.Contains(got, "-") {
		t.Errorf("empty footer should contain dashes, got: %q", got)
	}
}

func TestFooterRenderFull(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetDir("~/project")
	m.SetModel("gpt-4")
	m.SetTokens(1234, 10000)
	m.SetCost(0.05)

	got := m.Render()

	checks := []struct {
		name string
		want string
	}{
		{"directory", "~/project"},
		{"model", "gpt-4"},
		{"token count", "1234"},
		{"cost", "$0.05"},
	}

	for _, tt := range checks {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(got, tt.want) {
				t.Errorf("render should contain %q, got: %q", tt.want, got)
			}
		})
	}
}

func TestFooterRenderEmpty(t *testing.T) {
	m := NewFooterModel(testStyles())
	got := m.Render()

	// Empty state should show placeholder dashes
	if !strings.Contains(got, "—") {
		t.Errorf("empty footer should show dashes, got: %q", got)
	}
}

func TestFooterTokensPercent(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetTokens(1234, 10000)

	got := m.Render()

	// Should show percentage (1234/10000 = 12%)
	if !strings.Contains(got, "12%") {
		t.Errorf("footer should show 12%%, got: %q", got)
	}
}

func TestFooterCostZero(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetCost(0)

	got := m.Render()

	// Zero cost should show $0.00 or $0.00 placeholder
	if !strings.Contains(got, "$0") {
		t.Errorf("zero cost should show $0, got: %q", got)
	}
}

func TestFooterCostNonZero(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetCost(1.23)

	got := m.Render()

	if !strings.Contains(got, "$1.23") {
		t.Errorf("footer should show $1.23, got: %q", got)
	}
}

func TestFooterTheme(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewFooterModel(styles)
	m.SetModel("test")

	// Verify the footer was created with the theme styles
	got := m.Render()
	if got == "" {
		t.Error("themed footer should render non-empty")
	}
}
