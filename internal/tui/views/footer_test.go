package views

import (
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/tui/theme"
)

// Pi footer contract (REQ-TUI-APP-6): exactly 2 dim lines —
// L1 `{cwd} ({branch}) {session}`; L2 left stats, right `(provider) model`.
// Unknowns are omitted, never fabricated. There are no profile tabs, no
// welcome tick, no LSP/MCP dots (those live in /status now).

func TestNewFooterModel(t *testing.T) {
	m := NewFooterModel(testStyles())
	got := m.Render()
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Errorf("empty footer should render exactly 2 lines, got %d: %q", len(lines), got)
	}
	// Nothing known → nothing shown: no fabricated tokens, cost, branch,
	// session, provider, model, TAB hint or legacy welcome/dots.
	for _, f := range []string{"tokens", "$", "(", "TAB", "Get started", "/connect", "/status", "•", "⊙", "△"} {
		if strings.Contains(got, f) {
			t.Errorf("empty footer must omit unknowns, must not contain %q, got: %q", f, got)
		}
	}
}

func TestFooterLineContract(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetWidth(80)
	m.SetDir("/repo")
	m.SetBranch("main")
	m.SetSession("dev")
	m.SetTokens(1234, 10000)
	m.SetCost(0.05)
	m.SetProvider("openai")
	m.SetModel("gpt-4o")
	m.SetThinking("high")
	got := m.Render()
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("footer should render exactly 2 lines, got %d: %q", len(lines), got)
	}
	if !strings.Contains(lines[0], "/repo (main) dev") {
		t.Errorf("L1 should show cwd + (branch) + session, got: %q", lines[0])
	}
	if !strings.Contains(lines[1], "1,234 tokens 12%") {
		t.Errorf("L2 left should show honest stats, got: %q", lines[1])
	}
	if !strings.Contains(lines[1], "$0.05") {
		t.Errorf("L2 left should show cost once set, got: %q", lines[1])
	}
	if !strings.Contains(lines[1], "(openai) gpt-4o") {
		t.Errorf("L2 right should show (provider) model, got: %q", lines[1])
	}
	if !strings.Contains(lines[1], "high") {
		t.Errorf("L2 right should show thinking, got: %q", lines[1])
	}
}

func TestFooterUnknownsOmitted(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetDir("/repo")
	m.SetModel("gpt-4")
	// Legacy sync fields are compat no-ops: MCP/LSP/perm dots live in
	// /status now and must never leak into the footer.
	m.SetConnected(true)
	m.SetLSP(2)
	m.SetMCP(1)
	m.SetPerm(3)
	got := m.Render()
	for _, f := range []string{"•", "⊙", "△", "/status", "/connect", "Get started", "0"} {
		if strings.Contains(got, f) {
			t.Errorf("footer must omit legacy sync markers, must not contain %q, got: %q", f, got)
		}
	}
	if !strings.Contains(got, "/repo") {
		t.Errorf("footer should keep known dir, got: %q", got)
	}
	if !strings.Contains(got, "gpt-4") {
		t.Errorf("footer should keep known model, got: %q", got)
	}
}

func TestFooterTickIsStable(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetDir("/repo")
	before := m.Render()
	m.Tick()
	m.Tick()
	if got := m.Render(); got != before {
		t.Errorf("Tick is a compat no-op; footer must be stable, before %q after %q", before, got)
	}
}

func TestFooterTabHint(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetProfiles([]string{"coder", "writer"}, 0)
	if got := m.Render(); !strings.Contains(got, "TAB") {
		t.Errorf("footer with several profiles should hint TAB, got: %q", got)
	}
	single := NewFooterModel(testStyles())
	single.SetProfiles([]string{"coder"}, 0)
	if got := single.Render(); strings.Contains(got, "TAB") {
		t.Errorf("footer with one profile should omit TAB hint, got: %q", got)
	}
}

func TestFooterNoProfilesHint(t *testing.T) {
	// REQ-TUI-PROF-4: no discoverable profiles → muted hint, never crash.
	empty := NewFooterModel(testStyles())
	empty.SetDir("/repo")
	if got := empty.Render(); !strings.Contains(got, "no profiles available") {
		t.Errorf("footer with no profiles should hint, got: %q", got)
	}
	single := NewFooterModel(testStyles())
	single.SetDir("/repo")
	single.SetProfiles([]string{"coder"}, 0)
	if got := single.Render(); strings.Contains(got, "no profiles") {
		t.Errorf("footer with one profile should omit no-profiles hint, got: %q", got)
	}
	multi := NewFooterModel(testStyles())
	multi.SetDir("/repo")
	multi.SetProfiles([]string{"coder", "writer"}, 0)
	if got := multi.Render(); strings.Contains(got, "no profiles") {
		t.Errorf("footer with profiles should omit no-profiles hint, got: %q", got)
	}
}

func TestFooterTheme(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewFooterModel(styles)
	m.SetDir("/repo")
	m.SetModel("gpt-4")
	got := m.Render()
	if got == "" {
		t.Error("themed footer should render non-empty")
	}
}
