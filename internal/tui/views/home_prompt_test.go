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
	// Should contain placeholder text when empty
	if !strings.Contains(got, "Ask") {
		t.Errorf("prompt should contain placeholder 'Ask', got: %q", got)
	}
}

func TestHomePromptShowsPlaceholderWhenEmpty(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewHomePromptModel(styles)
	got := m.View(60)
	if !strings.Contains(got, "Ask kui") {
		t.Errorf("empty prompt should show 'Ask kui', got: %q", got)
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
	// After submit, value should be cleared
	if m.Value() != "" {
		t.Errorf("after Submit(), Value() = %q, want empty", m.Value())
	}
}
