package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// TestChatPerPartNoSplitBorder locks the pi contract (REQ-TUI-CHAT-1/2):
// user prompts render as UserMessageBg Boxes, assistant answers as plain
// markdown + Spacer(1). The old ┃/╹ SplitBorder language, ▣ end-caps,
// hover fill and QUEUED badge are gone and must never regress.
func TestChatPerPartNoSplitBorder(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "part one", "coder", "gpt-4")
	m.AppendMessage("user", "part two", "coder", "gpt-4")
	m.AppendMessage("assistant", "answer", "", "")
	got := m.View(80)
	for _, old := range []string{"┃", "╹", "▣", "QUEUED", "you:"} {
		if strings.Contains(got, old) {
			t.Errorf("chat must not contain removed marker %q, got:\n%s", old, got)
		}
	}
	if !strings.Contains(got, "part one") || !strings.Contains(got, "part two") {
		t.Errorf("chat should contain both user parts, got:\n%s", got)
	}
	if !strings.Contains(got, "answer") {
		t.Errorf("chat should contain the assistant answer, got:\n%s", got)
	}
}

// TestChatNoFullWidthDottedBand proves the ╹ end-cap never repeats as a
// full-width band (regression: lipgloss Border bottom rendered ╹ across the
// whole block width after every part).
func TestChatNoFullWidthDottedBand(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("assistant", "part one", "", "")
	m.AppendMessage("user", "part two", "coder", "gpt-4")
	got := m.View(80)
	for i, line := range strings.Split(got, "\n") {
		trimmed := strings.Trim(line, " ")
		if len(trimmed) > 3 && strings.Trim(trimmed, "╹") == "" {
			t.Errorf("line %d is a full-width ╹ band, want single end-cap char:\n%q", i, line)
		}
	}
}

func TestChatQueuedRendersAsNormalBox(t *testing.T) {
	// The QUEUED badge is REMOVED (REQ-TUI-CHAT-2, pi parity); queued
	// prompts render as ordinary user Boxes. AppendQueuedMessage stays
	// for API compat.
	m := NewChatModel(testStyles())
	m.AppendQueuedMessage("user", "hello", "coder", "gpt-4")
	got := m.View(80)
	if strings.Contains(got, "QUEUED") {
		t.Errorf("queued prompt must not carry a QUEUED badge, got:\n%s", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("queued prompt should render its content, got:\n%s", got)
	}
}

func TestChatHoverHasNoVisualEffect(t *testing.T) {
	// Hover fill is REMOVED (REQ-TUI-CHAT-2, pi parity); SetHover stays
	// for API compat and must not change rendering.
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "hello", "coder", "gpt-4")
	plain := m.View(80)
	m.SetHover(0, true)
	if got := m.View(80); got != plain {
		t.Errorf("hover must have no visual effect, plain:\n%s\nhovered:\n%s", plain, got)
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
	// Collapsed output shows a 10-line preview plus the pi expand hint
	// `… N lines` (REQ-TUI-TOOL-1); the 500-line body trims its trailing
	// newline to 500 lines, so 490 remain beyond the preview.
	if !strings.Contains(got, "\u2026 490 lines") {
		t.Errorf("collapsed output should render a \u2026 N lines expand hint, got: %q", got)
	}
	m.SetCollapse(false)
	got2 := m.Render()
	// Expanded large outputs upgrade to the indented panel block with its own hint.
	if got == got2 {
		t.Error("collapsed vs not collapsed should differ")
	}
}

func TestToolShowDetails(t *testing.T) {
	m := NewToolModel(testStyles())
	m.AppendCall("c1", "read_file")
	m.AppendResult("c1", "secret details")
	// showDetails toggles the call-id META only: result activity is the
	// point of this view and stays visible either way.
	m.SetShowDetails(false)
	got := m.Render()
	if strings.Contains(got, "(c1)") {
		t.Error("showDetails=false should hide the call-id meta")
	}
	if !strings.Contains(got, "secret details") {
		t.Error("result activity must stay visible regardless of showDetails")
	}
	m.SetShowDetails(true)
	got2 := m.Render()
	if !strings.Contains(got2, "(c1)") {
		t.Error("showDetails=true should show the call-id meta")
	}
}
