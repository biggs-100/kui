package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestFrameIsAPaintedCanvas proves the upstream root-box architecture: the
// whole terminal surface is painted with the theme background — no visible
// cell anywhere (spaces included) may fall back to the terminal default.
// This is what makes kui read as one big canvas containing nested fills,
// instead of floating elements over a foreign background.
func TestFrameIsAPaintedCanvas(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	for _, tc := range []struct {
		w, h  int
		route string
	}{{100, 30, "home"}, {100, 30, "session"}, {160, 30, "session"}} {
		app := newSessionApp(t, tc.w, tc.h)
		app.route = tc.route
		dump := stripTitleSequence(app.View())

		lines := strings.Split(dump, "\n")
		if len(lines) != tc.h {
			t.Fatalf("%dx%d %s: frame rows = %d, want %d", tc.w, tc.h, tc.route, len(lines), tc.h)
		}
		for i, line := range lines {
			if hole := firstCellWithoutTruecolorBG(line); hole >= 0 {
				t.Errorf("%dx%d %s row %d has an unpainted cell at byte %d:\n%q",
					tc.w, tc.h, tc.route, i, hole, line)
				break
			}
		}
	}
}

// firstCellWithoutTruecolorBG walks an ANSI row and returns the byte offset
// of the first cell (any glyph, spaces included) rendered without an active
// truecolor background sequence, or -1 when fully covered. Rows shorter than
// the terminal are padded by paintCanvas before this runs; the caller trims
// nothing, so any missing tail is caught too.
func firstCellWithoutTruecolorBG(row string) int {
	bgActive := false
	i := 0
	for i < len(row) {
		if row[i] == 0x1b {
			end := strings.IndexByte(row[i:], 'm')
			if end < 0 {
				return -1 // malformed; don't false-positive
			}
			params := strings.Split(strings.TrimSuffix(strings.TrimPrefix(row[i:i+end+1], "\x1b["), "m"), ";")
			hasReset := false
			bgSet := false
			for k := 0; k < len(params); k++ {
				switch params[k] {
				case "0", "":
					hasReset = true
				case "49":
					bgSet, hasReset = false, false
				case "48":
					bgSet = true // any truecolor/indexed bg paints the cell
				case "7":
					// reverse video swaps fg/bg: the cell gets painted
					bgSet = true
				}
			}
			switch {
			case len(params) == 1 && hasReset:
				bgActive = false
			case bgSet || (!hasReset && bgActive):
				bgActive = true
			default:
				bgActive = false
			}
			i += end + 1
			continue
		}
		// A byte of a visible cell: it must sit under an active bg. The first
		// uncovered byte of a run is reported; continuation bytes of a
		// multi-byte rune never get reached because we return immediately.
		if !bgActive {
			return i
		}
		i++
	}
	return -1
}
