package views

import (
	"strings"
	"testing"
)

func TestFooterLspStatus(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetLSPStatus("running", "")

	got := m.Render()
	if !strings.Contains(got, "LSP") {
		t.Errorf("footer should contain LSP status, got: %q", got)
	}
	if !strings.Contains(got, "running") {
		t.Errorf("footer should show 'running', got: %q", got)
	}
}

func TestFooterLspDisconnected(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetLSPStatus("", "")

	got := m.Render()
	// When no LSP status set, should show LSP: --
	if !strings.Contains(got, "LSP") {
		t.Errorf("footer should always show LSP section, got: %q", got)
	}
}

func TestFooterDiagnosticCount(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetLSPStatus("running", "3 errors, 2 warnings")

	got := m.Render()
	if !strings.Contains(got, "3 errors") {
		t.Errorf("footer should show error count, got: %q", got)
	}
	if !strings.Contains(got, "2 warnings") {
		t.Errorf("footer should show warning count, got: %q", got)
	}
}

func TestFooterDiagnosticCountZero(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetLSPStatus("running", "0 errors, 0 warnings")

	got := m.Render()
	if !strings.Contains(got, "LSP") {
		t.Errorf("footer should show LSP section, got: %q", got)
	}
}

func TestFooterFullLayout(t *testing.T) {
	m := NewFooterModel(testStyles())
	m.SetDir("~/project")
	m.SetModel("gpt-4")
	m.SetTokens(1234, 10000)
	m.SetCost(0.05)
	m.SetMCPStatus(2, 0)
	m.SetLSPStatus("running", "1 error, 0 warnings")

	got := m.Render()

	checks := []struct {
		name string
		want string
	}{
		{"directory", "~/project"},
		{"model", "gpt-4"},
		{"tokens", "1234"},
		{"cost", "$0.05"},
		{"MCP", "MCP"},
		{"LSP", "LSP"},
		{"lsp status", "running"},
		{"diagnostics", "1 error"},
	}

	for _, tt := range checks {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(got, tt.want) {
				t.Errorf("render should contain %q, got: %q", tt.want, got)
			}
		})
	}
}
