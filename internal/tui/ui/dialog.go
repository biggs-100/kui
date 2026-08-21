package ui

import (
	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

type Dialog struct {
	Size    int
	Content string
}

func NewDialog(size int, content string) Dialog { return Dialog{Size: size, Content: content} }

func (d Dialog) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return d.Content
	}
	size := d.Size
	if size <= 0 {
		size = 60
	}
	t := theme.DefaultTheme()
	topPad := height / 4
	boxHeight := height - 2*topPad
	if boxHeight < 5 {
		boxHeight = 5
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
