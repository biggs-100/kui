package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolCallsRender(t *testing.T) {
	m := NewToolModel()
	m.AppendCall("call-1", "read_file")
	m.AppendResult("call-1", "file contents here")
	got := m.Render()

	if !strings.Contains(got, "read_file") {
		t.Error("render should contain tool call name")
	}
	if !strings.Contains(got, "file contents here") {
		t.Error("render should contain tool result")
	}
}

func TestToolNilObserverEmptyList(t *testing.T) {
	m := NewToolModel()
	got := m.Render()
	// When no tool events are present, render should not be empty —
	// it should show an empty state hint
	if strings.TrimSpace(got) == "" {
		t.Error("tool view should show empty state, not empty string")
	}
}

func TestToolGoldenCallAndResult(t *testing.T) {
	m := NewToolModel()
	m.AppendCall("c1", "read_file")
	m.AppendResult("c1", "contents")
	got := m.Render()

	golden := filepath.Join("testdata", "tool_call_result.txt")
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
		t.Errorf("tool golden mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestToolGoldenEmpty(t *testing.T) {
	m := NewToolModel()
	got := m.Render()

	golden := filepath.Join("testdata", "tool_empty.txt")
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
		t.Errorf("tool empty golden mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestToolMultipleCalls(t *testing.T) {
	m := NewToolModel()
	m.AppendCall("c1", "read_file")
	m.AppendCall("c2", "write_file")
	m.AppendResult("c1", "done reading")
	m.AppendResult("c2", "done writing")
	got := m.Render()

	if !strings.Contains(got, "read_file") {
		t.Error("should contain read_file")
	}
	if !strings.Contains(got, "write_file") {
		t.Error("should contain write_file")
	}
	if !strings.Contains(got, "done reading") {
		t.Error("should contain result for read_file")
	}
	if !strings.Contains(got, "done writing") {
		t.Error("should contain result for write_file")
	}
}

func TestToolPendingCall(t *testing.T) {
	m := NewToolModel()
	m.AppendCall("c1", "bash_exec")
	got := m.Render()

	if !strings.Contains(got, "bash_exec") {
		t.Error("should contain tool call name")
	}
	// Before result, should show pending indicator
	if strings.Contains(got, "done") || strings.Contains(got, "result") {
		t.Error("should not show result for pending call")
	}
}
