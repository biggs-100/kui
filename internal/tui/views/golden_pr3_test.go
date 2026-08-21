package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/biggs-100/kui/internal/adapters/git"
	"github.com/biggs-100/kui/internal/tui/theme"
)

func chatGoldenWidth(t *testing.T, w int) {
	t.Helper()
	orig := ChatNow
	ChatNow = func() time.Time { return time.Date(2026, 8, 20, 14, 5, 0, 0, time.Local) }
	defer func() { ChatNow = orig }()
	m := NewChatModel(testStyles())
	m.SetWidth(w)
	m.AppendMessage("user", "hello", "coder", "gpt-4")
	m.AppendMessage("assistant", "# Title\n\nSome **bold** text with `code`.", "", "")
	m.AppendChunk(" plus streamed")
	m.AppendQueuedMessage("user", "queued prompt", "coder", "gpt-4")
	m.AppendPart(PartKindCompaction, "", "", "")
	got := m.View(w)
	golden := filepath.Join("testdata", "chat_"+itoa(w)+".txt")
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
		t.Fatalf("golden file not found (run with -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("chat golden %d mismatch\ngot:\n%s\nwant:\n%s", w, got, string(want))
	}
}

func itoa(n int) string {
	switch n {
	case 80:
		return "80"
	case 120:
		return "120"
	case 160:
		return "160"
	default:
		return "80"
	}
}

func TestChatGolden80(t *testing.T)  { chatGoldenWidth(t, 80) }
func TestChatGolden120(t *testing.T) { chatGoldenWidth(t, 120) }
func TestChatGolden160(t *testing.T) { chatGoldenWidth(t, 160) }

func TestDiffGoldenWidths(t *testing.T) {
	diffs := []git.FileDiff{
		{
			Path:      "main.go",
			Status:    "modified",
			Additions: 10,
			Deletions: 2,
			Hunks: []git.Hunk{
				{
					Header:   "@@ -10,7 +10,8 @@",
					OldStart: 10,
					NewStart: 10,
					Lines: []git.DiffLine{
						{Type: "context", Content: "package main", OldNum: 10, NewNum: 10},
						{Type: "removed", Content: "fmt.Println(\"hello\")", OldNum: 11, NewNum: 0},
						{Type: "added", Content: "fmt.Println(\"world\")", OldNum: 0, NewNum: 11},
						{Type: "added", Content: "fmt.Println(\"done\")", OldNum: 0, NewNum: 12},
						{Type: "context", Content: "}", OldNum: 12, NewNum: 13},
					},
				},
			},
		},
		{
			Path:      "helper.go",
			Status:    "added",
			Additions: 5,
			Deletions: 0,
			Hunks: []git.Hunk{
				{
					Header:   "@@ -0,0 +1,5 @@",
					OldStart: 0,
					NewStart: 1,
					Lines: []git.DiffLine{
						{Type: "added", Content: "package helper", OldNum: 0, NewNum: 1},
						{Type: "added", Content: "func Help() {}", OldNum: 0, NewNum: 2},
					},
				},
			},
		},
	}
	widths := []int{80, 120, 160}
	for _, w := range widths {
		m := NewDiffModel(testStyles())
		m.SetDiffs(diffs)
		m.SetWidth(w)
		m.SetWrapMode("word")
		got := m.View()
		golden := filepath.Join("testdata", "diff_"+itoa(w)+".txt")
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
			t.Fatalf("diff golden %d not found (run with -update): %v", w, err)
		}
		if got != string(want) {
			t.Errorf("diff golden %d mismatch\ngot:\n%s\nwant:\n%s", w, got, string(want))
		}
	}
}

func TestDiffWrapNone(t *testing.T) {
	m := NewDiffModel(testStyles())
	longContent := strings.Repeat("a", 200)
	diffs := []git.FileDiff{
		{
			Path:      "long.go",
			Status:    "modified",
			Additions: 1,
			Deletions: 0,
			Hunks: []git.Hunk{
				{
					Header:   "@@ -1,1 +1,1 @@",
					OldStart: 1,
					NewStart: 1,
					Lines: []git.DiffLine{
						{Type: "added", Content: longContent, OldNum: 0, NewNum: 1},
					},
				},
			},
		},
	}
	m.SetDiffs(diffs)
	m.SetWidth(80)
	m.SetWrapMode("none")
	got := m.View()
	lines := strings.Split(got, "\n")
	found := false
	for _, l := range lines {
		if strings.Contains(l, "a") && strings.Contains(l, "+") {
			if len(l) > 200 {
				t.Errorf("wrap none should truncate, got line length %d", len(l))
			}
			found = true
		}
	}
	if !found {
		t.Error("diff wrap none: expected long line in view")
	}
}

func TestSidebarLocale42(t *testing.T) {
	styles := theme.NewStyles(theme.DefaultTheme())
	m := NewSidebarModel(styles)
	m.SetTokens(1234567, 2000000)
	m.SetCost(12.34)
	m.SetProfile("coder")
	m.SetModel("gpt-4")
	m.SetTitle("My Session")
	m.SetSessionID("abc123")
	m.SetWorkspace("/tmp/project")
	got := m.View(42)
	if !strings.Contains(got, "1,234,567 tokens") {
		t.Errorf("sidebar should show locale formatted tokens, got: %q", got)
	}
	if !strings.Contains(got, "My Session") {
		t.Errorf("sidebar should contain title, got: %q", got)
	}
	if !strings.Contains(got, "abc123") {
		t.Errorf("sidebar should contain sessionID, got: %q", got)
	}
	if !strings.Contains(got, "$12.34") {
		t.Errorf("sidebar should contain cost, got: %q", got)
	}
	if got == "" {
		t.Error("sidebar view empty")
	}
}

func TestDiffLineNumbersStyled(t *testing.T) {
	m := NewDiffModel(testStyles())
	diffs := []git.FileDiff{
		{
			Path:      "main.go",
			Status:    "modified",
			Additions: 1,
			Deletions: 1,
			Hunks: []git.Hunk{
				{
					Header:   "@@ -10,2 +10,2 @@",
					OldStart: 10,
					NewStart: 10,
					Lines: []git.DiffLine{
						{Type: "context", Content: "package main", OldNum: 10, NewNum: 10},
						{Type: "removed", Content: "old", OldNum: 11, NewNum: 0},
						{Type: "added", Content: "new", OldNum: 0, NewNum: 11},
					},
				},
			},
		},
	}
	m.SetDiffs(diffs)
	got := m.View()
	if !strings.Contains(got, "▶") {
		t.Error("diff view should contain ▶ cursor")
	}
	if !strings.Contains(got, "+1") && !strings.Contains(got, "-1") {
		t.Error("diff view should contain +N/-N")
	}
	if !strings.Contains(got, "10") {
		t.Error("diff view should contain line numbers")
	}
}

func TestChatPerPartSplitBorder(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("assistant", "part one", "", "")
	m.AppendMessage("assistant", "part two", "", "")
	got := m.View(80)
	if !strings.Contains(got, "┃") {
		t.Error("chat per-part should contain ┃ left border")
	}
	if !strings.Contains(got, "╹") {
		t.Error("chat per-part should contain ╹ terminator")
	}
	// Ensure not using plain you: label
	if strings.Contains(got, "you:") {
		t.Error("chat should use border not you: label")
	}
}

func TestChatQueuedBadge(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendQueuedMessage("user", "hello", "coder", "gpt-4")
	got := m.View(80)
	if !strings.Contains(got, "QUEUED") {
		t.Error("queued badge should contain QUEUED")
	}
}

func TestChatHoverBackground(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "hello", "coder", "gpt-4")
	m.SetHover(0, true)
	got := m.View(80)
	// Should contain hover marker and backgroundElement path
	if !strings.Contains(got, "hover") {
		t.Error("hover should contain hover marker")
	}
}

func TestChatCompactionDivider(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("assistant", "hello", "", "")
	m.AppendPart(PartKindCompaction, "", "", "")
	got := m.View(80)
	if !strings.Contains(got, "compaction") {
		t.Error("compaction divider should contain compaction")
	}
}

func TestToolCollapse(t *testing.T) {
	m := NewToolModel(testStyles())
	m.AppendCall("c1", "read_file")
	long := strings.Repeat("line\n", 500)
	m.AppendResult("c1", long)
	m.SetCollapse(true)
	got := m.Render()
	if !strings.Contains(got, "…") || !strings.Contains(got, "lines") {
		t.Errorf("collapsed output should truncate with hint, got: %q", got)
	}
	m.SetCollapse(false)
	got2 := m.Render()
	// When not collapsed, should still contain content but maybe truncated differently
	if got == got2 {
		t.Error("collapsed vs not collapsed should differ")
	}
}

func TestToolShowDetails(t *testing.T) {
	m := NewToolModel(testStyles())
	m.AppendCall("c1", "read_file")
	m.AppendResult("c1", "secret details")
	m.SetShowDetails(false)
	got := m.Render()
	if strings.Contains(got, "secret details") {
		t.Error("showDetails=false should hide details")
	}
	m.SetShowDetails(true)
	got2 := m.Render()
	if !strings.Contains(got2, "secret details") {
		t.Error("showDetails=true should show details")
	}
}
