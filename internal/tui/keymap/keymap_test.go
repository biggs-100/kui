package keymap

import (
	"strings"
	"testing"
)

func TestKeymapLayers(t *testing.T) {
	if BaseLayer != "base" {
		t.Errorf("BaseLayer = %q, want base", BaseLayer)
	}
	if ModalLayer != "modal" {
		t.Errorf("ModalLayer = %q, want modal", ModalLayer)
	}
	// Stack should support push/pop
	km := New()
	if km.Current() != BaseLayer {
		t.Errorf("initial layer = %q, want base", km.Current())
	}
	km.Push(ModalLayer)
	if km.Current() != ModalLayer {
		t.Errorf("after push modal, current = %q, want modal", km.Current())
	}
	km.Pop()
	if km.Current() != BaseLayer {
		t.Errorf("after pop, current = %q, want base", km.Current())
	}
}

func TestFormatKeyBindings(t *testing.T) {
	// Leader + p should contain leader prefix
	got := FormatKeyBindings([]string{"leader", "p"})
	if got == "" {
		t.Fatal("FormatKeyBindings returned empty")
	}
	// Should contain leader token
	if !contains(got, "leader") && !contains(got, "Leader") && !contains(got, "<leader>") {
		// Allow alternative leader representation but must contain prefix
		t.Errorf("FormatKeyBindings(%q) = %q, should contain leader prefix", []string{"leader", "p"}, got)
	}
	// pgup alias should map to pgup (not pageup)
	got2 := FormatKeyBindings([]string{"pgup"})
	if !contains(got2, "pgup") {
		t.Errorf("FormatKeyBindings pgup = %q, want pgup", got2)
	}
	// Triangulate: multiple bindings
	got3 := FormatKeyBindings([]string{"ctrl+p"})
	if got3 == "" {
		t.Error("FormatKeyBindings ctrl+p empty")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestBindingsDeclaredInTable(t *testing.T) {
	// All bindings must be declared in table, not scattered
	bindings := AllBindings()
	if len(bindings) == 0 {
		t.Fatal("AllBindings returned empty, want declared bindings")
	}
	// Check that base bindings exist
	found := false
	for _, b := range bindings {
		if b.Layer == BaseLayer {
			found = true
			break
		}
	}
	if !found {
		t.Error("no base layer bindings found in table")
	}
}
