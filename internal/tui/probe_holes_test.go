package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/muesli/termenv"
	"github.com/charmbracelet/lipgloss"
)

var holeRe = regexp.MustCompile(`\x1b\[0m(?:\x1b\[[0-9;]*m)* {2,}`)

func TestProbeHoles(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	for _, tc := range []struct{ w, h int; route string }{
		{100, 30, "home"}, {140, 30, "session"}, {160, 30, "session"}, {100, 30, "session"},
	} {
		app := newSessionApp(t, tc.w, tc.h)
		app.switchTheme("opencode")
		app.route = tc.route
		dump := stripTitleSequence(app.View())
		for i, row := range strings.Split(dump, "\n") {
			if holeRe.MatchString(row) {
				m := holeRe.FindStringIndex(row)
				lo := m[0] - 20
				if lo < 0 { lo = 0 }
				hi := m[1] + 20
				if hi > len(row) { hi = len(row) }
				t.Logf("HOLE %dx%d %s row%d ctx=%q", tc.w, tc.h, tc.route, i, row[lo:hi])
			}
			if strings.Contains(row, "\x1b[40m") || strings.Contains(row, "\x1b[48;5;0m") {
				t.Logf("ANSI_BLACK %dx%d %s row%d", tc.w, tc.h, tc.route, i)
			}
		}
	}
}
