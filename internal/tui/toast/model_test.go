package toast

import (
	"strings"
	"testing"
	"time"

	"github.com/biggs-100/kui/internal/tui/theme"
)

func testStyles() *theme.Styles {
	return theme.NewStyles(theme.DefaultTheme())
}

func TestToastCreate(t *testing.T) {
	m := NewModel(testStyles())
	if m == nil {
		t.Fatal("NewModel returned nil")
	}
}

func TestToastPush(t *testing.T) {
	m := NewModel(testStyles())
	m.Push("saved config", LevelInfo, 3*time.Second)
	m.Push("error occurred", LevelError, 5*time.Second)

	toasts := m.Toasts()
	if len(toasts) != 2 {
		t.Fatalf("expected 2 toasts, got %d", len(toasts))
	}
	if toasts[0].Text != "saved config" {
		t.Errorf("toast[0].Text = %q, want %q", toasts[0].Text, "saved config")
	}
	if toasts[0].Level != LevelInfo {
		t.Errorf("toast[0].Level = %d, want %d", toasts[0].Level, LevelInfo)
	}
	if toasts[1].Text != "error occurred" {
		t.Errorf("toast[1].Text = %q, want %q", toasts[1].Text, "error occurred")
	}
	if toasts[1].Level != LevelError {
		t.Errorf("toast[1].Level = %d, want %d", toasts[1].Level, LevelError)
	}
}

func TestToastDismiss(t *testing.T) {
	m := NewModel(testStyles())
	// Push with zero duration — should expire immediately
	m.Push("expired", LevelInfo, 0)

	toasts := m.Toasts()
	if len(toasts) != 1 {
		t.Fatalf("expected 1 toast before tick, got %d", len(toasts))
	}

	// Simulate a tick — the toast should be dismissed
	updated, cmd := m.Update(TickMsg{})
	_ = cmd
	m = updated

	toasts = m.Toasts()
	if len(toasts) != 0 {
		t.Errorf("expected 0 toasts after tick, got %d", len(toasts))
	}
}

func TestToastRender(t *testing.T) {
	m := NewModel(testStyles())
	m.Push("info message", LevelInfo, 3*time.Second)
	m.Push("success message", LevelSuccess, 3*time.Second)
	m.Push("warning message", LevelWarn, 3*time.Second)
	m.Push("error message", LevelError, 3*time.Second)

	got := m.View()
	if got == "" {
		t.Fatal("View() returned empty string")
	}
	if !strings.Contains(got, "info message") {
		t.Error("View() should contain info message")
	}
	if !strings.Contains(got, "success message") {
		t.Error("View() should contain success message")
	}
	if !strings.Contains(got, "warning message") {
		t.Error("View() should contain warning message")
	}
	if !strings.Contains(got, "error message") {
		t.Error("View() should contain error message")
	}
}

func TestToastViewEmpty(t *testing.T) {
	m := NewModel(testStyles())
	got := m.View()
	if got != "" {
		t.Errorf("View() with no toasts should return empty, got %q", got)
	}
}

func TestToastLevelConstants(t *testing.T) {
	if LevelInfo != 0 {
		t.Errorf("LevelInfo = %d, want 0", LevelInfo)
	}
	if LevelSuccess != 1 {
		t.Errorf("LevelSuccess = %d, want 1", LevelSuccess)
	}
	if LevelWarn != 2 {
		t.Errorf("LevelWarn = %d, want 2", LevelWarn)
	}
	if LevelError != 3 {
		t.Errorf("LevelError = %d, want 3", LevelError)
	}
}

func TestToastMultipleTicksDismissExpired(t *testing.T) {
	m := NewModel(testStyles())
	// Push with zero duration
	m.Push("expires-1", LevelInfo, 0)
	m.Push("expires-2", LevelWarn, 0)
	// Push with long duration — should survive
	m.Push("persists", LevelSuccess, 10*time.Hour)

	// First tick: both zero-duration toasts expire
	updated, _ := m.Update(TickMsg{})
	m = updated

	toasts := m.Toasts()
	if len(toasts) != 1 {
		t.Fatalf("expected 1 toast after tick, got %d", len(toasts))
	}
	if toasts[0].Text != "persists" {
		t.Errorf("remaining toast = %q, want %q", toasts[0].Text, "persists")
	}
}
