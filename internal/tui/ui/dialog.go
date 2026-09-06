package ui

import (
	"strings"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

type Dialog struct {
	Size    int
	Content string
}

func NewDialog(size int, content string) Dialog { return Dialog{Size: size, Content: content} }

// NarrowSize clamps a preferred dialog width so the box fits narrow
// terminals (REQ-TUI-DLG-1): at most width-4 (margin for border/padding),
// floored at 20 so the box never collapses or panics.
func NarrowSize(preferred, width int) int {
	size := preferred
	if size <= 0 {
		size = 60
	}
	if width > 0 {
		if max := width - 4; size > max {
			size = max
		}
	}
	if size < 20 {
		size = 20
	}
	return size
}

// Rule returns a full-width ─ separator sized for a dialog content area of
// innerWidth columns (the DynamicBorder-style rule used between a dialog
// filter/title line and its list, estilo pi). Callers style it, e.g. with
// BorderSubtle.
func Rule(innerWidth int) string {
	if innerWidth < 10 {
		innerWidth = 10
	}
	return strings.Repeat("─", innerWidth)
}

func (d Dialog) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return d.Content
	}
	size := NarrowSize(d.Size, width)
	t := theme.DefaultTheme()
	topPad := height / 4
	boxHeight := height - 2*topPad
	if boxHeight < 5 {
		boxHeight = 5
	}
	if height > 4 && boxHeight > height-2 {
		boxHeight = height - 2
	}
	box := lipgloss.NewStyle().
		Width(size).
		Height(boxHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.BorderSubtle)).
		Background(lipgloss.Color(t.Background)).
		Padding(1, 1).
		Render(d.Content)
	centered := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
	backdrop := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Background(lipgloss.Color(OverlayBackdrop)).
		Render(centered)
	return backdrop
}

func (d Dialog) IsModal() bool { return true }
func (d Dialog) HandleKey(key string) bool {
	switch key {
	case "esc", "escape", "ctrl+c":
		return true
	}
	return false
}

const OverlayBackdrop = "rgba(0,0,0,150)"
