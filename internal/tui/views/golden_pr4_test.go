package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

func dialogGoldenPath(name string, w int) string {
	return filepath.Join("testdata", name+"_"+itoa(w)+".txt")
}

func TestDialogPaletteGolden120(t *testing.T) {
	cmds := []Command{
		{Name: "/sessions", Description: "List and manage sessions", Category: "Session", Suggested: true},
		{Name: "/model", Description: "Switch model", Category: "Model", Suggested: true},
		{Name: "/help", Description: "Show help", Category: "System", Suggested: true},
		{Name: "/reload", Description: "Hot-reload", Category: "Runtime"},
		{Name: "/status", Description: "Show status", Category: "Runtime"},
		{Name: "Ctrl+P", Description: "Command palette", Category: "Navigation", Shortcut: "Ctrl+P"},
	}
	m := NewCommandPaletteModel(cmds, 120, 30)
	m.SetStyles(testStyles())
	got := m.View()
	golden := dialogGoldenPath("dialog_palette", 120)
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden not found (run with -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("dialog_palette 120 mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
	if !strings.Contains(got, "Suggested") {
		t.Errorf("palette should contain Suggested header when no filter, got:\n%s", got)
	}
}

func TestDialogModelGolden120(t *testing.T) {
	models := []string{"gpt-4o", "claude-3.5-sonnet", "opencode/gpt-nano", "gemini-2.0-flash"}
	m := NewModelListModel(models, "gpt-4o", 120, 30)
	m.SetStyles(testStyles())
	got := m.View()
	golden := dialogGoldenPath("dialog_model", 120)
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden not found: %v", err)
	}
	if got != string(want) {
		t.Errorf("dialog_model 120 mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
	if !strings.Contains(got, "●") {
		t.Errorf("model dialog should contain current model dot ●, got:\n%s", got)
	}
	if !strings.Contains(got, "nano") {
		t.Errorf("model dialog should contain nano entry, got:\n%s", got)
	}
}

func TestDialogStatusGolden120(t *testing.T) {
	m := NewDialogStatusModel(120, 30)
	m.SetStyles(testStyles())
	m.SetMCP([]MCPServerInfo{
		{Name: "mcp1", Status: MCPConnected},
		{Name: "mcp2", Status: MCPFailed, Error: "connection refused"},
		{Name: "mcp3", Status: MCPDisabled},
		{Name: "mcp4", Status: MCPNeedsAuth},
	})
	m.SetLSP([]LSPServerInfo{
		{Name: "lsp1", Status: LSPConnected},
		{Name: "lsp2", Status: LSPFailed, Error: "timeout"},
	})
	m.SetFormatters([]FormatterInfo{{Name: "prettier", Source: "file://prettier"}})
	m.SetPlugins([]PluginInfo{{Name: "myplugin", Version: "1.0.0"}})
	got := m.View()
	golden := dialogGoldenPath("dialog_status", 120)
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden not found: %v", err)
	}
	if got != string(want) {
		t.Errorf("dialog_status 120 mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
	if !strings.Contains(got, "•") {
		t.Errorf("status should contain • dot, got:\n%s", got)
	}
	if !strings.Contains(got, "connection refused") {
		t.Errorf("status should contain error detail, got:\n%s", got)
	}
}

func TestDialogSessionGolden120(t *testing.T) {
	metas := []core.SessionMeta{
		{ID: "s1", Profile: "coder", Name: strings.Repeat("a", 90), CreatedAt: "2026-08-20"},
		{ID: "s2", Profile: "writer", Name: "short", CreatedAt: "2026-08-19"},
	}
	m := NewSessionListModel(metas, 120, 30)
	m.SetStyles(testStyles())
	got := m.View()
	golden := dialogGoldenPath("dialog_session", 120)
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden not found: %v", err)
	}
	if got != string(want) {
		t.Errorf("dialog_session 120 mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
	if !strings.Contains(got, "...") {
		t.Errorf("session dialog should truncate 76 with ..., got:\n%s", got)
	}
}
