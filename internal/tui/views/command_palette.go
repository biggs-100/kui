package views

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// Command holds the metadata for a single command displayed in the palette.
type Command struct {
	Name        string
	Description string
	Category    string
	Shortcut    string
	Args        string
}

// commandItem wraps a Command for display in the bubbles list.
type commandItem struct {
	cmd Command
}

func (i commandItem) FilterValue() string {
	return i.cmd.Name + " " + i.cmd.Description
}

// commandItemDelegate renders a single command entry in the palette list.
type commandItemDelegate struct{}

func (d commandItemDelegate) Height() int                             { return 1 }
func (d commandItemDelegate) Spacing() int                            { return 0 }
func (d commandItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d commandItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ci, ok := item.(commandItem)
	if !ok {
		return
	}

	name := ci.cmd.Name
	desc := ci.cmd.Description
	shortcut := ci.cmd.Shortcut

	// Truncate name and description for display
	if len(name) > 20 {
		name = name[:17] + "..."
	}
	if len(desc) > 40 {
		desc = desc[:37] + "..."
	}

	var line string
	if shortcut != "" {
		line = fmt.Sprintf("%-20s  %-40s  %-8s", name, desc, shortcut)
	} else {
		line = fmt.Sprintf("%-20s  %-40s", name, desc)
	}

	if index == m.Index() {
		_, _ = fmt.Fprintf(w, "▸ %s", line)
	} else {
		_, _ = fmt.Fprintf(w, "  %s", line)
	}
}

// CommandPaletteModel wraps a bubbles/list.Model for interactive command
// selection with fuzzy filtering.
type CommandPaletteModel struct {
	list        list.Model
	commands    []Command
	allItems    []commandItem
	filtered    []commandItem
	filterText  string
	selected    string
	quitting    bool
	width       int
	height      int
}

// NewCommandPaletteModel creates a CommandPaletteModel from a slice of Commands.
func NewCommandPaletteModel(cmds []Command, width, height int) CommandPaletteModel {
	items := make([]list.Item, len(cmds))
	allItems := make([]commandItem, len(cmds))
	for i, cmd := range cmds {
		allItems[i] = commandItem{cmd: cmd}
		items[i] = allItems[i]
	}

	l := list.New(items, commandItemDelegate{}, width, height)
	l.Title = "Command Palette"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // we handle filtering ourselves
	l.Styles.Title = lipgloss.NewStyle().Bold(true)
	l.SetShowHelp(false)

	return CommandPaletteModel{
		list:     l,
		commands: cmds,
		allItems: allItems,
		filtered: allItems,
		width:    width,
		height:   height,
	}
}

// applyFilter updates the list items based on the current filter text.
func (m *CommandPaletteModel) applyFilter() {
	var matched []commandItem
	if m.filterText == "" {
		matched = m.allItems
	} else {
		matched = fuzzyMatchItems(m.filterText, m.allItems)
	}
	m.filtered = matched

	items := make([]list.Item, len(matched))
	for i, item := range matched {
		items[i] = item
	}
	m.list.SetItems(items)
}

// fuzzyMatchItems performs fuzzy matching of query against command items.
func fuzzyMatchItems(query string, items []commandItem) []commandItem {
	targets := make([]string, len(items))
	for i, item := range items {
		targets[i] = item.cmd.Name + " " + item.cmd.Description
	}

	matches := fuzzy.Find(query, targets)
	result := make([]commandItem, 0, len(matches))
	for _, match := range matches {
		result = append(result, items[match.Index])
	}
	return result
}

// Update handles keyboard input for the command palette.
func (m CommandPaletteModel) Update(msg tea.Msg) (CommandPaletteModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if idx := m.list.Index(); idx >= 0 && idx < len(m.filtered) {
				m.selected = m.filtered[idx].cmd.Name
			}
			return m, tea.Quit
		case tea.KeyEscape:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyBackspace:
			if len(m.filterText) > 0 {
				m.filterText = m.filterText[:len(m.filterText)-1]
				m.applyFilter()
			}
			return m, nil
		case tea.KeyRunes:
			m.filterText += string(msg.Runes)
			m.applyFilter()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the command palette.
func (m CommandPaletteModel) View() string {
	if m.quitting {
		return ""
	}

	// Show filter input line
	var filterLine string
	if m.filterText != "" {
		filterLine = fmt.Sprintf("  > %s_\n", m.filterText)
	} else {
		filterLine = "  > _\n"
	}

	return "\n" + filterLine + m.list.View()
}

// Selected returns the name of the command the user selected, or empty string
// if the user dismissed the palette.
func (m CommandPaletteModel) Selected() string {
	return m.selected
}

// FuzzyMatch performs fuzzy matching of a query against a list of commands,
// returning matched commands ranked by score.
func FuzzyMatch(query string, cmds []Command) []Command {
	if query == "" {
		return cmds
	}

	targets := make([]string, len(cmds))
	for i, cmd := range cmds {
		targets[i] = cmd.Name + " " + cmd.Description
	}

	matches := fuzzy.Find(query, targets)
	result := make([]Command, 0, len(matches))
	for _, match := range matches {
		result = append(result, cmds[match.Index])
	}
	return result
}
