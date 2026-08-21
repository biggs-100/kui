package views

import (
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/tui/theme"
)

func TestHomePromptRendersWithBorder(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	got := m.View(60)
	if got == "" {
		t.Error("HomePromptModel View() returned empty string")
	}
	// Should contain placeholder text from pool when empty
	found := false
	for _, ph := range placeholderPool {
		if strings.Contains(got, ph) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("prompt should contain placeholder from pool, got: %q", got)
	}
	// Border must be SplitBorder with decorative bottom ▀
	if !strings.Contains(got, "▀") {
		t.Errorf("prompt should contain decorative bottom '▀', got: %q", got)
	}
	if !strings.Contains(got, "┃") && !strings.Contains(got, "╹") {
		t.Errorf("prompt should contain SplitBorder '┃' or '╹', got: %q", got)
	}
}

func TestHomePromptShowsPlaceholderWhenEmpty(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	got := m.View(60)
	found := false
	for _, ph := range placeholderPool {
		if strings.Contains(got, ph) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("empty prompt should show placeholder from pool, got: %q", got)
	}
}

func TestHomePromptAcceptsInput(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	m.SetValue("hello")
	got := m.View(60)
	if !strings.Contains(got, "hello") {
		t.Errorf("prompt should contain typed text, got: %q", got)
	}
}

func TestHomePromptSetValue(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	m.SetValue("test")
	if m.Value() != "test" {
		t.Errorf("Value() = %q, want %q", m.Value(), "test")
	}
}

func TestHomePromptClear(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	m.SetValue("hello")
	m.Clear()
	if m.Value() != "" {
		t.Errorf("after Clear(), Value() = %q, want empty", m.Value())
	}
}

func TestHomePromptSubmit(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	m.SetValue("test prompt")
	got := m.Submit()
	if got != "test prompt" {
		t.Errorf("Submit() = %q, want %q", got, "test prompt")
	}
	if m.Value() != "" {
		t.Errorf("after Submit(), Value() = %q, want empty", m.Value())
	}
}

func TestHomePromptMaxWidth75AtWide(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	got := m.View(160)
	// At 160 cols, prompt width should be 75 (capped), not 70
	// Check that rendered width is ~75 by checking that prompt contains decorative strip of 77 (75+2)
	// Decorative is 77 "▀" chars (promptWidth+2)
	if !strings.Contains(got, strings.Repeat("▀", 77)) {
		t.Errorf("prompt at 160 should have width 75 (decorative 77), got: %q", got)
	}
}

func TestHomePromptAuto70AtNarrow(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	got := m.View(80)
	// At 80 cols, 70% is 56, decorative should be 58
	if !strings.Contains(got, strings.Repeat("▀", 58)) {
		t.Errorf("prompt at 80 should have width ~56 (decorative 58), got: %q", got)
	}
}

func TestHomePromptPlaceholderPool(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		got := m.View(60)
		for _, ph := range placeholderPool {
			if strings.Contains(got, ph) {
				seen[ph] = true
			}
		}
	}
	if len(seen) < 2 {
		t.Errorf("placeholder should vary across pool, saw only %d distinct placeholders: %v", len(seen), seen)
	}
}

func TestHomePromptShellMode(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	m.SetValue("!ls")
	got := m.View(60)
	if !strings.Contains(got, "!") {
		t.Errorf("shell mode prompt should contain '!', got: %q", got)
	}
	if !strings.Contains(got, "!ls") {
		t.Errorf("shell mode should contain '!ls', got: %q", got)
	}
}

func TestHomePromptExtmarks(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"file", "check file.go", "● [File]"},
		{"image", "show image", "● [Image]"},
		{"pasted", "pasted content", "● [Pasted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewHomePromptModel(styles)
			m.SetValue(tt.input)
			got := m.View(60)
			if !strings.Contains(got, tt.want) {
				t.Errorf("extmark for %q should contain %q, got: %q", tt.input, tt.want, got)
			}
		})
	}
}

func TestHomePromptMaxHeight(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	m.SetHeight(30)
	if m.MaxHeight() != 10 {
		t.Errorf("MaxHeight for height 30 should be 10, got %d", m.MaxHeight())
	}
	m.SetHeight(15)
	if m.MaxHeight() != 6 {
		t.Errorf("MaxHeight for height 15 should be 6 (max 6), got %d", m.MaxHeight())
	}
	m.SetHeight(0)
	if m.MaxHeight() != 6 {
		t.Errorf("MaxHeight for height 0 should be 6, got %d", m.MaxHeight())
	}
}
