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

		// Full-row walk: no VISIBLE (non-space) cell may render without the
		// panel background active. This catches every segment whose style
		// forgot its own bg — inner resets kill the container's fill.
		if hole := firstUnpaintedCell(line, r, g, b); hole >= 0 {
			t.Errorf("row %d renders a visible cell without panel background at byte %d:\n%q", i, hole, line)
		}
	}
}

// firstUnpaintedCell walks an ANSI row tracking the active background across
// full SGR sequences (combined fg+bg runs included). It returns the byte
// offset of the first printable non-space cell whose active background is
// not the panel color, or -1 when every visible cell is covered.
func firstUnpaintedCell(row string, wr, wg, wb int64) int {
	want := fmt.Sprintf("48;2;%d;%d;%d", wr, wg, wb)
	bgActive := false
	i := 0
	for i < len(row) {
		if row[i] == 0x1b {
			end := strings.IndexByte(row[i:], 'm')
			if end < 0 {
				return -1 // malformed; don't false-positive
			}
			params := strings.Split(strings.TrimSuffix(strings.TrimPrefix(row[i:i+end+1], "\x1b["), "m"), ";")
			bgSet := false
			hasReset := false
			for k := 0; k < len(params); k++ {
				switch params[k] {
				case "0", "":
					hasReset = true
				case "49":
					bgSet, hasReset = false, false // default bg explicitly set
				case "48":
					if k+1 < len(params) && params[k+1] == "2" && k+5 < len(params) {
						bgSet = strings.Join(params[k:k+5], ";") == want
					} else {
						bgSet = true // indexed/other bg: terminal paints something
					}
				}
			}
			switch {
			case len(params) == 1 && hasReset:
				bgActive = false
			default:
				if bgSet || (!hasReset && bgActive) {
					bgActive = true
				} else {
					bgActive = bgSet
				}
			}
			i += end + 1
			continue
		}
		ch := row[i]
		if ch != ' ' && !bgActive {
			return i
		}
		i++
	}
	return -1
}
