package util

import "github.com/charmbracelet/lipgloss"

// TruncateMiddle truncates s to max width by keeping start and end with ... in middle.
// Uses lipgloss.Width for correct display width.
func TruncateMiddle(s string, max int) string {
	if max <= 0 {
		return s
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	// Need to keep width, not bytes; use runes.
	runes := []rune(s)
	// Estimate half widths.
	leftKeep := (max - 3) / 2
	rightKeep := max - 3 - leftKeep
	// Adjust for display width differences (simplified: assume 1 width per rune for non-ansi)
	if len(runes) <= leftKeep+rightKeep+3 {
		return string(runes[:max-3]) + "..."
	}
	// Build left part by width
	left := ""
	w := 0
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if w+rw > leftKeep {
			break
		}
		left += string(r)
		w += rw
	}
	// Build right part from end
	right := ""
	w = 0
	for i := len(runes) - 1; i >= 0; i-- {
		r := runes[i]
		rw := lipgloss.Width(string(r))
		if w+rw > rightKeep {
			break
		}
		right = string(r) + right
		w += rw
	}
	return left + "..." + right
}
