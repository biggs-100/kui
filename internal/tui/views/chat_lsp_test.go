package views

import (
	"strings"
	"testing"
)

func TestChatDiagnosticAnnotation(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "Show me main.go", "coder", "gpt-4")
	m.AppendChunk("Here is the file content")

	// Add diagnostic annotation below the last message.
	m.SetDiagnostics([]string{
		"ERROR: undefined: foo (line 5)",
		"WARNING: unused import (line 2)",
	})

	got := m.Render()
	if !strings.Contains(got, "ERROR: undefined: foo") {
		t.Errorf("render should contain diagnostic ERROR, got: %q", got)
	}
	if !strings.Contains(got, "WARNING: unused import") {
		t.Errorf("render should contain diagnostic WARNING, got: %q", got)
	}
}

func TestChatNoDiagnostics(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "Hello", "coder", "gpt-4")

	got := m.Render()
	// No diagnostics set — should not contain diagnostic markers.
	if strings.Contains(got, "ERROR:") || strings.Contains(got, "WARNING:") {
		t.Errorf("render should not contain diagnostic markers when none set, got: %q", got)
	}
}

func TestChatClearDiagnostics(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "Hello", "coder", "gpt-4")
	m.SetDiagnostics([]string{"ERROR: test"})
	m.SetDiagnostics(nil)

	got := m.Render()
	if strings.Contains(got, "ERROR: test") {
		t.Errorf("render should not contain cleared diagnostics, got: %q", got)
	}
}

func TestChatDiagnosticCount(t *testing.T) {
	m := NewChatModel(testStyles())
	m.SetDiagnostics([]string{
		"ERROR: one",
		"ERROR: two",
		"WARNING: three",
	})

	diags := m.Diagnostics()
	if len(diags) != 3 {
		t.Errorf("Diagnostics() returned %d, want 3", len(diags))
	}
}
