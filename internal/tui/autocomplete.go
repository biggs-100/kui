package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/views"
)

// AutocompleteItem is a single completion entry with display metadata.
type AutocompleteItem struct {
	Value       string
	Label       string
	Description string
}

// AutocompleteModel manages intelligent completion for slash commands, arg values, and file @ mentions.
type AutocompleteModel struct {
	commands []string
	filtered []string
	index    int
	active   bool

	maxVisible  int
	items       []AutocompleteItem // last filtered items (detailed)
	prefix      string
	allCommands []AutocompleteItem // command items for fuzzy filtering
}

// NewAutocompleteModel creates an AutocompleteModel with default commands derived from the registry.
func NewAutocompleteModel() AutocompleteModel {
	registry := NewCommandRegistry()
	cmds := registry.CommandNames()
	all := make([]AutocompleteItem, 0, len(registry.All()))
	for _, c := range registry.All() {
		if !strings.HasPrefix(c.Name, "/") {
			continue
		}
		label := c.Name
		desc := c.Description
		if c.Args != "" {
			desc = desc + " " + c.Args
		}
		all = append(all, AutocompleteItem{Value: c.Name, Label: label, Description: desc})
	}
	return AutocompleteModel{
		commands:    cmds,
		allCommands: all,
		maxVisible:  5,
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
	a.items = nil
	a.index = 0
	a.prefix = ""
}

// IsActive returns whether the autocomplete popup is showing.
func (a AutocompleteModel) IsActive() bool {
	return a.active
}

// Filter updates the filtered list based on the input prefix using fuzzy matching.
// Supports slash command, arg, file @ completions, Shell ! mode and ●File extmarks.
func (a *AutocompleteModel) Filter(input string) {
	a.prefix = input
	trimmed := strings.TrimSpace(input)

	// Shell ! mode: input starting with "!" at offset 0 triggers Warning border and file completions with ●File extmarks
	if strings.HasPrefix(trimmed, "!") {
		shellPrefix := strings.TrimSpace(strings.TrimPrefix(trimmed, "!"))
		items := fileCompletions(shellPrefix)
		if len(items) > 0 || shellPrefix == "" {
			if shellPrefix == "" {
				items = fileCompletions("")
			}
			// Transform to shell items with ●File extmark and without @ prefix for insertion after "!"
			shellItems := make([]AutocompleteItem, len(items))
			for i, it := range items {
				v := strings.TrimPrefix(it.Value, "@")
				shellItems[i] = AutocompleteItem{Value: v, Label: it.Label, Description: "●File"}
			}
			a.items = shellItems
			a.filtered = make([]string, len(shellItems))
			for i, it := range shellItems {
				a.filtered[i] = it.Value
			}
			if len(a.filtered) == 0 {
				a.Deactivate()
				return
			}
			if a.index >= len(a.filtered) {
				a.index = 0
			}
			a.active = true
			return
		}
	}

	// File @ completion: if input contains "@" before cursor, prioritize file suggestions.
	// Extract prefix after last "@" with no intervening space.
	if atPrefix, ok := extractAtPrefix(input); ok {
		// If we're at an @ mention, show file completions.
		// Use fuzzy over file list with prefix.
		items := fileCompletions(atPrefix)
		// If no files matched but prefix empty, fall through to slash handling? Keep files empty to deactivate.
		if len(items) > 0 || atPrefix == "" {
			// When atPrefix is empty (just "@"), show top files
			if atPrefix == "" {
				items = fileCompletions("")
			}
			a.items = items
			a.filtered = make([]string, len(items))
			for i, it := range items {
				a.filtered[i] = it.Value
			}
			if len(a.filtered) == 0 {
				a.Deactivate()
				return
			}
			if a.index >= len(a.filtered) {
				a.index = 0
			}
			a.active = true
			return
		}
	}

	// Arg completion: input like "/model <arg>" or "/login <arg>"
	if strings.HasPrefix(trimmed, "/") && strings.Contains(trimmed, " ") {
		parts := strings.Fields(trimmed)
		if len(parts) >= 1 {
			cmdName := parts[0]
			// Extract arg prefix: text after first space, trimmed
			argPrefix := ""
			if idx := strings.Index(trimmed, " "); idx >= 0 {
				argPrefix = strings.TrimSpace(trimmed[idx+1:])
				// If there are multiple args, use last token for completion
				if fields := strings.Fields(argPrefix); len(fields) > 1 {
					argPrefix = fields[len(fields)-1]
				}
			}
			if items := argumentCompletions(cmdName, argPrefix); items != nil {
				// fuzzy filter already applied inside argumentCompletions
				a.items = items
				a.filtered = make([]string, len(items))
				for i, it := range items {
					a.filtered[i] = it.Value
				}
				if len(a.filtered) == 0 {
					a.Deactivate()
					return
				}
				if a.index >= len(a.filtered) {
					a.index = 0
				}
				a.active = true
				return
			}
		}
	}

	// Slash command completion: text before cursor starts with "/" and contains no space.
	if strings.HasPrefix(trimmed, "/") && !strings.Contains(trimmed, " ") {
		// "/" alone should show all slash commands without fuzzy filtering.
		if trimmed == "/" {
			a.items = a.allCommands
			a.filtered = make([]string, len(a.allCommands))
			for i, it := range a.allCommands {
				a.filtered[i] = it.Value
			}
			if len(a.filtered) == 0 {
				a.Deactivate()
				return
			}
			if a.index >= len(a.filtered) {
				a.index = 0
			}
			a.active = true
			return
		}
		// fuzzy over command items using the full trimmed as query
		matched := fuzzyFilter(trimmed, a.allCommands, func(it AutocompleteItem) string { return it.Value })
		// Also try filtering on label+description? Use Value for primary but include description in search via composite
		// For better help matching, also consider description fuzzy via fallback if no value match
		if len(matched) == 0 {
			// try matching against label+description
			matched = fuzzyFilter(trimmed, a.allCommands, func(it AutocompleteItem) string { return it.Label + " " + it.Description })
		}
		a.items = matched
		a.filtered = make([]string, len(matched))
		for i, it := range matched {
			a.filtered[i] = it.Value
		}
		if len(a.filtered) == 0 {
			a.Deactivate()
			return
		}
		if a.index >= len(a.filtered) {
			a.index = 0
		}
		a.active = true
		return
	}

	// No applicable completion -> deactivate
	a.Deactivate()
}

// Selected returns the currently selected command value.
func (a AutocompleteModel) Selected() string {
	if len(a.filtered) == 0 {
		return ""
	}
	if a.index < 0 || a.index >= len(a.filtered) {
		return ""
	}
	return a.filtered[a.index]
}

// SelectedItem returns the currently selected AutocompleteItem, or nil if none.
func (a AutocompleteModel) SelectedItem() *AutocompleteItem {
	if len(a.items) == 0 || a.index < 0 || a.index >= len(a.items) {
		return nil
	}
	it := a.items[a.index]
	return &it
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
// Handles slash, arg, file @ replacements, Shell ! mode and ●File extmarks.
func (a *AutocompleteModel) Accept(input string) string {
	selected := a.Selected()
	a.Deactivate()
	if selected == "" {
		return input
	}

	// Shell ! mode: input starts with "!" and selected is file path without @
	if strings.HasPrefix(strings.TrimSpace(input), "!") {
		if idx := strings.Index(input, "!"); idx >= 0 {
			prefix := input[:idx+1]
			// trim after "!" and replace with selected
			// Keep "!" plus selected
			if strings.TrimSpace(input[idx+1:]) == "" {
				return prefix + selected
			}
			// Replace last token after "!"
			suffix := strings.TrimPrefix(input[idx+1:], " ")
			lastSpace := strings.LastIndex(suffix, " ")
			if lastSpace >= 0 {
				return prefix + suffix[:lastSpace+1] + selected
			}
			return prefix + selected
		}
	}

	// File @ replacement: if selected starts with "@" and input contains "@"
	if strings.HasPrefix(selected, "@") && strings.Contains(input, "@") {
		if idx := strings.LastIndex(input, "@"); idx >= 0 {
			return input[:idx] + selected
		}
	}

	// Arg completion: input contains space and selected is not a slash command
	if strings.Contains(strings.TrimSpace(input), " ") {
		trimmed := strings.TrimSpace(input)
		// If selected is a slash command, replace first token (unlikely in arg mode)
		if strings.HasPrefix(selected, "/") {
			parts := strings.Fields(input)
			if len(parts) > 0 && strings.HasPrefix(parts[len(parts)-1], "/") {
				parts[len(parts)-1] = selected
				return strings.Join(parts, " ")
			}
			return selected
		}
		// Normal arg: replace last arg token
		spaceIdx := strings.LastIndex(input, " ")
		if spaceIdx >= 0 {
			// Preserve prefix including space
			return input[:spaceIdx+1] + selected
		}
		_ = trimmed
		return selected
	}

	// Slash command replacement
	words := strings.Fields(input)
	if len(words) == 0 {
		return selected
	}
	lastWord := words[len(words)-1]
	if strings.HasPrefix(lastWord, "/") || strings.HasPrefix(lastWord, "@") {
		words[len(words)-1] = selected
	} else {
		words = append(words, selected)
	}
	return strings.Join(words, " ")
}

// View renders the autocomplete popup as a string, limited to maxVisible.
func (a AutocompleteModel) View() string {
	if !a.active || len(a.filtered) == 0 {
		return ""
	}
	maxVis := a.maxVisible
	if maxVis <= 0 {
		maxVis = 5
	}
	if maxVis < 3 {
		maxVis = 3
	}
	if maxVis > 20 {
		maxVis = 20
	}

	// Prefer detailed items if available
	if len(a.items) > 0 && len(a.items) == len(a.filtered) {
		// Compute window around index
		start := 0
		if len(a.items) > maxVis {
			if a.index >= maxVis {
				start = a.index - maxVis + 1
			}
			if start+maxVis > len(a.items) {
				start = len(a.items) - maxVis
			}
			if start < 0 {
				start = 0
			}
		}
		end := start + maxVis
		if end > len(a.items) {
			end = len(a.items)
		}
		var b strings.Builder
		for i := start; i < end; i++ {
			it := a.items[i]
			label := it.Label
			if label == "" {
				label = it.Value
			}
			line := label
			if it.Description != "" {
				line = fmt.Sprintf("%s — %s", label, it.Description)
			}
			prefix := "  "
			if i == a.index {
				prefix = "> "
			}
			b.WriteString(prefix + line)
			if i < end-1 {
				b.WriteString("\n")
			}
		}
		if len(a.items) > maxVis {
			b.WriteString(fmt.Sprintf("\n  ... %d more", len(a.items)-maxVis))
		}
		return b.String()
	}

	// Fallback: plain filtered strings
	limit := maxVis
	if len(a.filtered) < limit {
		limit = len(a.filtered)
	}
	// Window around index for fallback too
	start := 0
	if len(a.filtered) > maxVis {
		if a.index >= maxVis {
			start = a.index - maxVis + 1
		}
		if start+maxVis > len(a.filtered) {
			start = len(a.filtered) - maxVis
		}
		if start < 0 {
			start = 0
		}
	}
	end := start + maxVis
	if end > len(a.filtered) {
		end = len(a.filtered)
	}
	var b strings.Builder
	for i := start; i < end; i++ {
		cmd := a.filtered[i]
		prefix := "  "
		if i == a.index {
			prefix = "> "
		}
		b.WriteString(prefix + cmd)
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	if len(a.filtered) > maxVis {
		b.WriteString(fmt.Sprintf("\n  ... %d more", len(a.filtered)-maxVis))
	}
	return b.String()
}

// argumentCompletions returns fuzzy-filtered arg suggestions for known slash commands.
func argumentCompletions(cmdName, argPrefix string) []AutocompleteItem {
	switch cmdName {
	case "/model":
		models := views.AvailableModelsFiltered()
		if len(models) == 0 {
			models = views.AvailableModels()
		}
		items := make([]AutocompleteItem, 0, len(models))
		for _, m := range models {
			// variant handling: include provider prefix variant if model contains "/"
			label := m
			desc := ""
			// ●File style extmark for file-like models? Not needed
			// variant handling: if argPrefix contains "/", suggest with provider prefix
			if strings.Contains(m, "/") {
				desc = "variant"
			}
			items = append(items, AutocompleteItem{Value: m, Label: label, Description: desc})
		}
		if argPrefix == "" {
			return items
		}
		return fuzzyFilter(argPrefix, items, func(it AutocompleteItem) string { return it.Label + " " + it.Description })
	case "/login", "/logout":
		providers := availableProviders()
		if argPrefix == "" {
			return providers
		}
		return fuzzyFilter(argPrefix, providers, func(it AutocompleteItem) string {
			// search combines id + label + description like Pi: id+name+authTypes
			return it.Value + " " + it.Label + " " + it.Description
		})
	case "/theme":
		names := theme.ThemeNames()
		if len(names) == 0 {
			names = []string{"kui-default", "opencode"}
		}
		// Include next/prev variants
		variants := []AutocompleteItem{
			{Value: "next", Label: "next", Description: "next theme"},
			{Value: "prev", Label: "prev", Description: "previous theme"},
		}
		for _, n := range names {
			variants = append(variants, AutocompleteItem{Value: n, Label: n, Description: "theme"})
		}
		if argPrefix == "" {
			return variants
		}
		return fuzzyFilter(argPrefix, variants, func(it AutocompleteItem) string { return it.Label })
	case "/sessions", "/resume", "/rename":
		// slash arg variants for sessions etc: no arg completions needed, but keep shell handling
		return nil
	default:
		return nil
	}
}

// availableProviders returns provider completion options similar to Pi getLoginProviderCompletionOptions.
func availableProviders() []AutocompleteItem {
	return []AutocompleteItem{
		{Value: "openai", Label: "openai", Description: "OpenAI · API Key"},
		{Value: "anthropic", Label: "anthropic", Description: "Anthropic · OAuth/API Key"},
		{Value: "opencode", Label: "opencode", Description: "Opencode · API Key"},
		{Value: "opencode-go", Label: "opencode-go", Description: "Opencode Go · API Key"},
		{Value: "gemini", Label: "gemini", Description: "Gemini · API Key"},
	}
}

// extractAtPrefix extracts the prefix after the last "@" before cursor.
// Returns (prefix, found). Found is true if "@" exists and suffix contains no space.
func extractAtPrefix(input string) (string, bool) {
	idx := strings.LastIndex(input, "@")
	if idx == -1 {
		return "", false
	}
	suffix := input[idx+1:]
	// If suffix contains space, the @ token is not the last token -> not a file mention
	if strings.Contains(suffix, " ") {
		return "", false
	}
	// Also ensure prefix doesn't contain control? Allow "/" and "." and alphanum-
	return suffix, true
}

// fileCompletions returns fuzzy-matched file suggestions for prefix via filesystem walk.
// Falls back to simple filepath.Walk with max 100 results and top 20.
func fileCompletions(prefix string) []AutocompleteItem {
	// Collect files via walk, max 100
	var paths []string
	root := "."
	// Use os.Getwd as base if available
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		root = cwd
	}
	count := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if count >= 100 {
			return filepath.SkipAll
		}
		// Exclude .git
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		// Skip hidden .kui? Keep hidden but exclude .git only as per spec includes --hidden
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		if rel == "." {
			return nil
		}
		// Normalize to forward slashes for display
		rel = filepath.ToSlash(rel)
		paths = append(paths, rel)
		count++
		return nil
	})

	// If prefix empty, return top 20 sorted alphabetically
	if prefix == "" {
		sort.Strings(paths)
		limit := 20
		if len(paths) < limit {
			limit = len(paths)
		}
		items := make([]AutocompleteItem, 0, limit)
		for i := 0; i < limit; i++ {
			p := paths[i]
			items = append(items, AutocompleteItem{Value: "@" + p, Label: p, Description: "●File"})
		}
		return items
	}

	type scored struct {
		path  string
		score int
	}
	var scoredPaths []scored
	for _, p := range paths {
		ok, score := fuzzyMatch(prefix, p)
		if ok {
			scoredPaths = append(scoredPaths, scored{path: p, score: score})
		}
	}
	sort.Slice(scoredPaths, func(i, j int) bool {
		if scoredPaths[i].score != scoredPaths[j].score {
			return scoredPaths[i].score < scoredPaths[j].score
		}
		return scoredPaths[i].path < scoredPaths[j].path
	})
	limit := 20
	if len(scoredPaths) < limit {
		limit = len(scoredPaths)
	}
	items := make([]AutocompleteItem, 0, limit)
	for i := 0; i < limit; i++ {
		p := scoredPaths[i].path
		items = append(items, AutocompleteItem{Value: "@" + p, Label: p, Description: "●File"})
	}
	return items
}
