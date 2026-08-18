package tui

import (
	"strings"
)

// defaultCommands is the set of slash commands available for autocomplete.
var defaultCommands = []string{
	"/reload",
	"/sessions",
	"/resume",
	"/rename",
	"/undo",
	"/redo",
	"/quit",
	"/exit",
	"/help",
	"/theme",
	"/status",
	"/clear",
}

// AutocompleteModel manages slash-command completion.
type AutocompleteModel struct {
	commands []string
	filtered []string
	index    int
	active   bool
}

// NewAutocompleteModel creates an AutocompleteModel with default commands.
func NewAutocompleteModel() AutocompleteModel {
	return AutocompleteModel{
		commands: defaultCommands,
	}
}

// Activate filters commands by prefix and shows the popup.
func (a *AutocompleteModel) Activate(input string) {
	a.index = 0
	a.active = true
	a.Filter(input)
}

// Deactivate hides the autocomplete popup.
func (a *AutocompleteModel) Deactivate() {
	a.active = false
	a.filtered = nil
	a.index = 0
}

// IsActive returns whether the autocomplete popup is showing.
func (a AutocompleteModel) IsActive() bool {
	return a.active
}

// Filter updates the filtered list based on the input prefix.
func (a *AutocompleteModel) Filter(input string) {
	prefix := strings.ToLower(input)
	a.filtered = a.filtered[:0]
	for _, cmd := range a.commands {
		if strings.HasPrefix(strings.ToLower(cmd), prefix) {
			a.filtered = append(a.filtered, cmd)
		}
	}
	if len(a.filtered) == 0 {
		a.Deactivate()
		return
	}
	if a.index >= len(a.filtered) {
		a.index = 0
	}
}

// Selected returns the currently selected command.
func (a AutocompleteModel) Selected() string {
	if len(a.filtered) == 0 {
		return ""
	}
	return a.filtered[a.index]
}

// MoveUp moves the selection up, wrapping to the bottom.
func (a *AutocompleteModel) MoveUp() {
	if len(a.filtered) == 0 {
		return
	}
	a.index--
	if a.index < 0 {
		a.index = len(a.filtered) - 1
	}
}

// MoveDown moves the selection down, wrapping to the top.
func (a *AutocompleteModel) MoveDown() {
	if len(a.filtered) == 0 {
		return
	}
	a.index++
	if a.index >= len(a.filtered) {
		a.index = 0
	}
}

// Accept selects the current item, replaces the partial input, and hides the popup.
// It finds the partial token (last word) starting with "/" and replaces it.
func (a *AutocompleteModel) Accept(input string) string {
	selected := a.Selected()
	a.Deactivate()
	if selected == "" {
		return input
	}

	// Find the last "/" token in input and replace it
	words := strings.Fields(input)
	if len(words) == 0 {
		return selected
	}
	lastWord := words[len(words)-1]
	if strings.HasPrefix(lastWord, "/") {
		words[len(words)-1] = selected
	} else {
		words = append(words, selected)
	}
	return strings.Join(words, " ")
}

// View renders the autocomplete popup as a string.
func (a AutocompleteModel) View() string {
	if !a.active || len(a.filtered) == 0 {
		return ""
	}
	var b strings.Builder
	for i, cmd := range a.filtered {
		if i == a.index {
			b.WriteString("> " + cmd)
		} else {
			b.WriteString("  " + cmd)
		}
		if i < len(a.filtered)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
