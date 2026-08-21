package views

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// stripANSI removes SGR escape sequences so tests can assert on visible text.
func stripANSI(s string) string {
	re := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return re.ReplaceAllString(s, "")
}

// TestSidebarFullHeightPinsFooter proves REQ-TUI-APP-2 full-height rail:
// GIVEN a target height WHEN ViewFullHeight renders THEN the block is exactly
// height rows wide-padded to 42, rendered as a PURE background panel (no
// border anywhere — elevation comes from the fill), with the workspace path
// pinned inside directly above the bottom padding row.
func TestSidebarFullHeightPinsFooter(t *testing.T) {
	m := NewSidebarModel(testStyles())
	m.SetTokens(1234, 10000)
	m.SetCost(0.05)
	m.SetProfile("coder")
	m.SetTitle("my session")
	m.SetWorkspace("~/dev-biggz/kui")

	const height = 24
	got := m.ViewFullHeight(42, height)
	lines := strings.Split(got, "\n")
	if len(lines) != height {
		t.Fatalf("ViewFullHeight should render exactly %d lines, got %d", height, len(lines))
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != 42 {
			t.Errorf("line %d visible width = %d, want 42", i, w)
		}
	}

	// Border-free contract: no box-drawing glyphs anywhere in the rail —
	// elevation comes from the backgroundPanel fill alone.
	for i, l := range lines {
		plain := stripANSI(l)
		if strings.ContainsAny(plain, "╭╰╮╯│┌┐└┘─") {
			t.Errorf("line %d contains border glyphs but the rail must be a pure background panel: %q", i, plain)
		}
	}

	// Footer pinned inside the padding: workspace path is the row above the
	// bottom padding row. When a buildinfo version exists it renders between
	// the path and that row.
	aboveBottom := stripANSI(lines[len(lines)-2])
	if !strings.Contains(aboveBottom, "~/dev-biggz/kui") {
		t.Errorf("row above the bottom padding should be the workspace path, got %q", aboveBottom)
	}

	// Blank filler in the middle keeps the rail visually continuous. Under
	// `go test` the color profile is Ascii (no TTY) and lipgloss strips
	// every escape, so the background is proven in
	// TestSidebarFullHeightBackgroundStrip with TrueColor forced.
	mid := stripANSI(lines[height/2])
	if strings.TrimSpace(mid) != "" {
		t.Errorf("middle filler line should be visually blank, got %q", mid)
	}
}

// TestSidebarFullHeightBackgroundStrip proves the filler rows carry the
// Sidebar background: with TrueColor forced, lipgloss styles its width-fill
// with the style background, so blank rows must contain an SGR sequence.
func TestSidebarFullHeightBackgroundStrip(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	m := NewSidebarModel(testStyles())
	m.SetWorkspace("~/dev-biggz/kui")
	const height = 20
	got := m.ViewFullHeight(42, height)
	lines := strings.Split(got, "\n")
	if len(lines) != height {
		t.Fatalf("ViewFullHeight should render exactly %d lines, got %d", height, len(lines))
	}
	mid := lines[height/2]
	if !strings.Contains(mid, "\x1b[48;2;") {
		t.Errorf("middle filler line lacks background SGR sequence (unstyled gap): %q", mid)
	}
	if w := lipgloss.Width(mid); w != 42 {
		t.Errorf("middle filler visible width = %d, want 42", w)
	}
}

// TestSidebarFooterNotAvailableWhenWorkspaceEmpty proves REQ-TUI-CHAT-6:
// absent workspace renders the muted NotAvailable placeholder, never a
// fabricated path.
func TestSidebarFooterNotAvailableWhenWorkspaceEmpty(t *testing.T) {
	m := NewSidebarModel(testStyles())
	got := stripANSI(m.View(42))
	if !strings.Contains(got, "NotAvailable") {
		t.Errorf("sidebar footer should show NotAvailable when workspace absent, got %q", got)
	}
}

// TestSidebarViewNaturalHeight keeps the narrow-overlay contract: plain View
// renders content-height only and still ends with the footer lines.
func TestSidebarViewNaturalHeight(t *testing.T) {
	m := NewSidebarModel(testStyles())
	m.SetWorkspace("~/dev-biggz/kui")
	got := m.View(42)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	last := stripANSI(lines[len(lines)-1])
	if !strings.Contains(last, "~/dev-biggz/kui") {
		t.Errorf("View should end with the workspace path footer, got %q", last)
	}
}
