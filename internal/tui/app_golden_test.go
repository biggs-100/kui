package tui

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

var updateAppGolden = flag.Bool("update-app-golden", false, "update app layout golden files")

// TestAppGoldenLayout proves REQ-TUI-APP-10 "Goldens lock layout": the full
// app View() dump is locked at 80/120/160 widths and every rendered line
// stays within the terminal column count (+1 tolerance per spec).
func TestAppGoldenLayout(t *testing.T) {
	for _, width := range []int{80, 120, 160} {
		t.Run(fmt.Sprintf("w%d", width), func(t *testing.T) {
			app := newSessionApp(t, width, 30)
			got := stripTitleSequence(app.View())
			golden := filepath.Join("testdata", fmt.Sprintf("app_%d.txt", width))
			if *updateAppGolden {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("golden not found (run with -update-app-golden): %v", err)
			}
			if got != string(want) {
				t.Errorf("app golden %d mismatch\ngot:\n%s\nwant:\n%s", width, got, string(want))
			}
			for i, line := range strings.Split(got, "\n") {
				if w := lipgloss.Width(line); w > width+1 {
					t.Errorf("line %d visible width %d exceeds %d (+1 tolerance)", i, w, width)
				}
			}
		})
	}
}
