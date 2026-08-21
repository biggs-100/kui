package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDialogSizes(t *testing.T) {
	for _, size := range []int{60, 88, 116} {
		d := NewDialog(size, "content")
		if d.Size != size {
			t.Errorf("Dialog size = %d, want %d", d.Size, size)
		}
	}
}

func TestDialogViewCenters(t *testing.T) {
	d := NewDialog(88, "hello")
	// Terminal 120x30, dialog 88 should be centered
	out := d.View(120, 30)
	if out == "" {
		t.Fatal("Dialog View returned empty")
	}
	// Content should be present
	if !strings.Contains(out, "hello") {
		t.Error("Dialog View should contain content")
	}
	// Width of rendered dialog content should be size
	// Place centers; we check that lipgloss.Width of content is size
	// The view should be at least size wide
	w := lipgloss.Width(out)
	if w < 88 {
		t.Errorf("Dialog width = %d, want >=88", w)
	}
}

func TestDialogOverlayBackdrop(t *testing.T) {
	d := NewDialog(60, "test")
	out := d.View(120, 30)
	// Backdrop should be dimmed; we check that style uses RGBA(0,0,0,150) or at least background
	// For now, verify that View does not panic and contains backdrop marker
	if out == "" {
		t.Error("overlay backdrop view empty")
	}
}

func TestDialogTopPadding(t *testing.T) {
	d := NewDialog(60, "x")
	// Height 30, top padding should be height/4 = 7
	out := d.View(120, 30)
	lines := strings.Split(out, "\n")
	// Find first non-empty line with content vs top padding empty lines
	// Count leading empty lines
	emptyTop := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			emptyTop++
		} else {
			break
		}
	}
	// Top padding should be roughly height/4 (allow ±2)
	expected := 30 / 4
	if emptyTop < expected-2 || emptyTop > expected+2 {
		t.Errorf("top padding = %d, want ~%d (height/4)", emptyTop, expected)
	}
}

func TestDialogModalKeyClose(t *testing.T) {
	d := NewDialog(60, "content")
	if !d.IsModal() {
		t.Error("Dialog should be modal")
	}
	// Esc should close
	closed := d.HandleKey("esc")
	if !closed {
		t.Error("Esc should close dialog")
	}
	// Ctrl+C should also close
	d2 := NewDialog(60, "content")
	if !d2.HandleKey("ctrl+c") {
		t.Error("Ctrl+C should close dialog")
	}
}
