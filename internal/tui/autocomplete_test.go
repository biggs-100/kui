package tui

import (
	"strings"
	"testing"
)

func TestAutocompleteCreate(t *testing.T) {
	a := NewAutocompleteModel()
	if a.IsActive() {
		t.Fatal("new autocomplete should not be active before Activate")
	}
	if len(a.commands) == 0 {
		t.Fatal("expected default commands to be loaded")
	}
}

func TestAutocompleteFilter(t *testing.T) {
	a := NewAutocompleteModel()
	a.Activate("/he")

	if !a.IsActive() {
		t.Fatal("autocomplete should be active after Activate")
	}
	if len(a.filtered) == 0 {
		t.Fatal("expected filtered results for /he")
	}
	// Should match /help
	found := false
	for _, c := range a.filtered {
		if c == "/help" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected /help in filtered results, got %v", a.filtered)
	}
}

func TestAutocompleteFilterNoMatch(t *testing.T) {
	a := NewAutocompleteModel()
	a.Activate("/xyz")

	// No matches — should deactivate or show empty
	if len(a.filtered) != 0 {
		t.Errorf("expected no matches for /xyz, got %v", a.filtered)
	}
}

func TestAutocompleteNav(t *testing.T) {
	a := NewAutocompleteModel()
	a.Activate("/")

	if len(a.filtered) < 2 {
		t.Skip("need at least 2 commands for nav test")
	}

	// Initial selection should be first item
	first := a.Selected()

	a.MoveDown()
	if a.Selected() == first {
		t.Error("MoveDown should change selection")
	}

	a.MoveUp()
	if a.Selected() != first {
		t.Error("MoveUp should return to first item")
	}
}

func TestAutocompleteNavWraps(t *testing.T) {
	a := NewAutocompleteModel()
	a.Activate("/")

	if len(a.filtered) == 0 {
		t.Skip("no commands available")
	}

	// MoveDown past end should wrap to first (full cycle)
	first := a.filtered[0]
	for i := 0; i < len(a.filtered); i++ {
		a.MoveDown()
	}
	if a.Selected() != first {
		t.Errorf("MoveDown should wrap, got %q want %q", a.Selected(), first)
	}
}

func TestAutocompleteAccept(t *testing.T) {
	a := NewAutocompleteModel()
	a.Activate("/he")

	if len(a.filtered) == 0 {
		t.Fatal("expected at least one match for /he")
	}

	result := a.Accept("hello /he world")
	// Should replace the partial input with the completed command
	if !strings.Contains(result, "/help") {
		t.Errorf("expected result to contain /help, got %q", result)
	}
	if a.IsActive() {
		t.Error("Accept should deactivate autocomplete")
	}
}

func TestAutocompleteDismiss(t *testing.T) {
	a := NewAutocompleteModel()
	a.Activate("/h")
	if !a.IsActive() {
		t.Fatal("should be active")
	}

	a.Deactivate()
	if a.IsActive() {
		t.Error("Deactivate should hide autocomplete")
	}
}

func TestAutocompleteView(t *testing.T) {
	a := NewAutocompleteModel()
	a.Activate("/")

	view := a.View()
	if view == "" {
		t.Error("expected non-empty view when active")
	}

	a.Deactivate()
	view = a.View()
	if view != "" {
		t.Error("expected empty view when inactive")
	}
}

func TestAutocompleteSelectedIndex(t *testing.T) {
	a := NewAutocompleteModel()
	a.Activate("/")

	if len(a.filtered) == 0 {
		t.Skip("no commands available")
	}

	// Should start at index 0
	if a.index != 0 {
		t.Errorf("initial index = %d, want 0", a.index)
	}

	a.MoveDown()
	if a.index != 1 {
		t.Errorf("index after MoveDown = %d, want 1", a.index)
	}

	a.MoveUp()
	if a.index != 0 {
		t.Errorf("index after MoveUp = %d, want 0", a.index)
	}
}
