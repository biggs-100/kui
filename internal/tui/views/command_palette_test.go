package views_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/biggs-100/kui/internal/tui/views"
)

// testCommands returns a standard set of commands for palette tests.
func testCommands() []views.Command {
	return []views.Command{
		{Name: "/sessions", Description: "List and manage sessions", Category: "Session"},
		{Name: "/resume", Description: "Resume a saved session", Category: "Session", Args: "<session-id>"},
		{Name: "/rename", Description: "Rename the current session", Category: "Session", Args: "<name>"},
		{Name: "/undo", Description: "Undo last conversation turn", Category: "Edit"},
		{Name: "/redo", Description: "Redo last undone turn", Category: "Edit"},
		{Name: "/clear", Description: "Clear chat display", Category: "Edit"},
		{Name: "/reload", Description: "Hot-reload runtime state", Category: "Runtime"},
		{Name: "/theme", Description: "Switch UI theme", Category: "Runtime", Args: "<name>"},
		{Name: "/status", Description: "Show current profile status", Category: "Runtime"},
		{Name: "/quit", Description: "Save and exit", Category: "System", Shortcut: "Ctrl+C"},
		{Name: "/exit", Description: "Save and exit", Category: "System"},
		{Name: "/help", Description: "Show this help", Category: "System"},
		{Name: "Tab", Description: "Switch profile", Category: "Navigation", Shortcut: "Tab"},
		{Name: "d", Description: "Toggle diff view", Category: "Navigation", Shortcut: "d"},
		{Name: "Ctrl+P", Description: "Command palette", Category: "Navigation", Shortcut: "Ctrl+P"},
	}
}

// TestPaletteCreate verifies that NewCommandPaletteModel creates a palette
// with all provided commands and correct initial state.
func TestPaletteCreate(t *testing.T) {
	cmds := testCommands()
	palette := views.NewCommandPaletteModel(cmds, 80, 24)

	// View should render without panic
	view := palette.View()
	if view == "" {
		t.Error("View() returned empty string")
	}

	// Initially nothing selected
	if palette.Selected() != "" {
		t.Errorf("Selected() = %q, want empty string on fresh model", palette.Selected())
	}
}

// TestPaletteCreateEmpty verifies that an empty command list produces a
// valid palette.
func TestPaletteCreateEmpty(t *testing.T) {
	palette := views.NewCommandPaletteModel(nil, 80, 24)

	view := palette.View()
	if view == "" {
		t.Error("View() returned empty string for empty palette")
	}
}

// TestPaletteFilter verifies that typing in the palette filters commands
// by fuzzy matching on name and description.
func TestPaletteFilter(t *testing.T) {
	cmds := testCommands()
	palette := views.NewCommandPaletteModel(cmds, 80, 24)

	// Type "reload" — should narrow to reload command
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	// View should contain "reload"
	view := palette.View()
	if !strings.Contains(view, "reload") {
		t.Errorf("palette view should contain 'reload' after filtering, got:\n%s", view)
	}

	// View should NOT contain session commands
	if strings.Contains(view, "/sessions") {
		t.Errorf("palette view should not contain '/sessions' when filtered to 'reload', got:\n%s", view)
	}
}

// TestPaletteEscape verifies that pressing Escape returns empty selection.
func TestPaletteEscape(t *testing.T) {
	cmds := testCommands()
	palette := views.NewCommandPaletteModel(cmds, 80, 24)

	// Press Escape
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if palette.Selected() != "" {
		t.Errorf("Selected() = %q, want empty after escape", palette.Selected())
	}
}

// TestPaletteEnter verifies that pressing Enter returns the selected command name.
func TestPaletteEnter(t *testing.T) {
	cmds := testCommands()
	palette := views.NewCommandPaletteModel(cmds, 80, 24)

	// Press Enter — should select the first item
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyEnter})

	selected := palette.Selected()
	if selected == "" {
		t.Fatal("Selected() returned empty after Enter, expected a command name")
	}
}

// TestPaletteNavigation verifies arrow key navigation.
func TestPaletteNavigation(t *testing.T) {
	cmds := testCommands()
	palette := views.NewCommandPaletteModel(cmds, 80, 24)

	// Down moves to next item
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyDown})
	// Enter selects the second item
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyEnter})

	selected := palette.Selected()
	if selected == "" {
		t.Fatal("Selected() returned empty after down+enter")
	}
	// Should not be the first command
	first := cmds[0].Name
	if selected == first {
		t.Errorf("after down+enter, selected = %q, should not be first command %q", selected, first)
	}
}

// TestPaletteFilterByDescription verifies fuzzy matching on description text.
func TestPaletteFilterByDescription(t *testing.T) {
	cmds := testCommands()
	palette := views.NewCommandPaletteModel(cmds, 80, 24)

	// Type "switch" — should match "Switch profile" (Tab command description)
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	palette, _ = palette.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	view := palette.View()
	if !strings.Contains(view, "Switch") && !strings.Contains(view, "switch") {
		t.Errorf("palette view should contain 'Switch' when filtered to 'switch', got:\n%s", view)
	}
}

// TestFuzzyMatch verifies the FuzzyMatch helper function.
func TestFuzzyMatch(t *testing.T) {
	cmds := testCommands()

	// Empty query returns all
	all := views.FuzzyMatch("", cmds)
	if len(all) != len(cmds) {
		t.Errorf("empty query returned %d results, want %d", len(all), len(cmds))
	}

	// Fuzzy match "reload"
	matched := views.FuzzyMatch("reload", cmds)
	if len(matched) == 0 {
		t.Fatal("FuzzyMatch('reload') returned 0 results")
	}
	if matched[0].Name != "/reload" {
		t.Errorf("FuzzyMatch('reload')[0].Name = %q, want /reload", matched[0].Name)
	}

	// Fuzzy match "sw" — should match "Switch profile"
	matched = views.FuzzyMatch("sw", cmds)
	if len(matched) == 0 {
		t.Fatal("FuzzyMatch('sw') returned 0 results")
	}
	found := false
	for _, cmd := range matched {
		if cmd.Name == "Tab" {
			found = true
			break
		}
	}
	if !found {
		t.Error("FuzzyMatch('sw') should include Tab (Switch profile)")
	}
}
