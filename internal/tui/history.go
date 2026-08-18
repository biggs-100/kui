package tui

import (
	"bufio"
	"os"
	"strings"
)

const maxHistoryEntries = 50

// History manages prompt history with JSONL persistence.
type History struct {
	path           string
	entries        []string
	index          int  // current position (-1 = at end, not browsing)
	atBeginning    bool // true when Previous() has been called past the first entry
}

// NewHistory creates a History from a JSONL file path.
// If the file does not exist, an empty history is returned.
func NewHistory(path string) (*History, error) {
	h := &History{
		path:  path,
		index: -1,
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return h, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			h.entries = append(h.entries, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return h, nil
}

// Append adds a prompt to history. Consecutive duplicates are removed.
// History is capped at maxHistoryEntries.
func (h *History) Append(prompt string) error {
	if prompt == "" {
		return nil
	}

	// Dedup consecutive: remove the last entry if it matches
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == prompt {
		h.entries = h.entries[:len(h.entries)-1]
	}

	h.entries = append(h.entries, prompt)

	// Trim to max entries
	if len(h.entries) > maxHistoryEntries {
		h.entries = h.entries[len(h.entries)-maxHistoryEntries:]
	}

	// Reset navigation to end
	h.index = -1
	h.atBeginning = false

	return h.save()
}

// Previous returns the previous prompt (up arrow).
// Returns empty string when past the beginning.
func (h *History) Previous() string {
	if len(h.entries) == 0 {
		return ""
	}

	if h.atBeginning {
		return ""
	}

	if h.index == -1 {
		// Not browsing: jump to last entry
		h.index = len(h.entries) - 1
	} else if h.index > 0 {
		h.index--
	} else {
		// At first entry (index == 0): next call will be past beginning
		h.atBeginning = true
	}

	return h.entries[h.index]
}

// Next returns the next prompt (down arrow).
// Returns empty string if at the end.
func (h *History) Next() string {
	if h.index == -1 {
		return ""
	}

	if h.atBeginning {
		// Past beginning: jump back to first entry
		h.atBeginning = false
		h.index = 0
		return h.entries[h.index]
	}

	h.index++
	if h.index >= len(h.entries) {
		h.index = -1
		return ""
	}

	return h.entries[h.index]
}

// Reset resets navigation to the end.
func (h *History) Reset() {
	h.index = -1
	h.atBeginning = false
}

// save writes all entries to the JSONL file.
func (h *History) save() error {
	var b strings.Builder
	for _, entry := range h.entries {
		b.WriteString(entry)
		b.WriteByte('\n')
	}
	return os.WriteFile(h.path, []byte(b.String()), 0o644)
}
