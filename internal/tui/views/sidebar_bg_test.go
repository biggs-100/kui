package views

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestSidebarRowsEndWithPanelBackground proves the transparency regression:
// inner styles must never use .Width() inside the rail, because lipgloss
// emits a Width-style's padding OUTSIDE its SGR run — punching unpainted
// terminal-default holes through the panel fill. Under TrueColor every rail
// row must terminate with a background-painted run matching BackgroundPanel.
func TestSidebarRowsEndWithPanelBackground(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	m := NewSidebarModel(testStyles())
	m.SetProfile("coder")
	m.SetTitle("my session")
	m.SetTokens(56, 0)
	m.SetWorkspace("~/dev-biggz/kui")

	got := m.ViewFullHeight(42, 18)

	hex := strings.TrimPrefix(m.styles.Theme.BackgroundPanel, "#")
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	bgSeq := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)

	lines := strings.Split(got, "\n")
	if len(lines) != 18 {
		t.Fatalf("rail height = %d, want 18", len(lines))
	}
	for i, line := range lines {
		if !strings.HasSuffix(line, "\x1b[0m") {
			t.Errorf("row %d does not end with a reset:\n%q", i, line)
			continue
		}
		tail := strings.TrimSuffix(line, "\x1b[0m")
		j := strings.LastIndex(tail, bgSeq)
		if j < 0 {
			t.Errorf("row %d has no panel-background run at its end:\n%q", i, line)
			continue
		}
		after := tail[j+len(bgSeq):]
		if strings.Trim(after, " ") != "" {
			t.Errorf("row %d has non-space content after the final background run:\n%q", i, line)
		}
	}
}
