package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

func TestChatMessageListGrowsOnChunk(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "Hello", "coder", "gpt-4")
	m.AppendChunk("Hi there!")

	got := m.Render()
	if !strings.Contains(got, "Hello") {
		t.Error("render should contain user message")
	}
	if !strings.Contains(got, "Hi there!") {
		t.Error("render should contain streamed chunk")
	}
}

func TestChatEmptyInputIgnored(t *testing.T) {
	m := NewChatModel(testStyles())
	// Render with no messages — should not panic and should not contain user content
	got := m.Render()
	if strings.Contains(got, "user") {
		t.Error("empty chat should not contain 'user' label")
	}
}

func TestChatStreamError(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "Hello", "coder", "gpt-4")
	m.SetError("provider timeout")

	got := m.Render()
	if !strings.Contains(got, "Hello") {
		t.Error("render should contain user message")
	}
	if !strings.Contains(got, "error") && !strings.Contains(got, "Error") {
		t.Error("render should contain error state")
	}
}

func TestChatGoldenDefault(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "What is 2+2?", "coder", "gpt-4")
	m.AppendChunk("4")
	got := m.Render()

	golden := filepath.Join("testdata", "chat_with_message.txt")
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
		t.Errorf("chat golden mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestChatGoldenError(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "Hello", "coder", "gpt-4")
	m.SetError("stream failed")
	got := m.Render()

	golden := filepath.Join("testdata", "chat_error_state.txt")
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
		t.Errorf("chat error golden mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestChatGoldenEmpty(t *testing.T) {
	m := NewChatModel(testStyles())
	got := m.Render()

	golden := filepath.Join("testdata", "chat_empty.txt")
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
		t.Errorf("chat empty golden mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestChatMultipleChunksGrowMessage(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "Explain Go", "coder", "gpt-4")
	m.AppendChunk("Go is ")
	m.AppendChunk("a language.")
	got := m.Render()
	if !strings.Contains(got, "Go is a language.") {
		t.Error("render should contain concatenated chunks")
	}
}

func TestChatPerPromptContextStability(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "First", "coder", "gpt-4")
	m.AppendChunk("Answer 1")
	m.AppendMessage("user", "Second", "writer", "gpt-3.5")
	m.AppendChunk("Answer 2")
	got := m.Render()

	if !strings.Contains(got, "coder") {
		t.Error("first message should show coder profile")
	}
	if !strings.Contains(got, "writer") {
		t.Error("second message should show writer profile")
	}
}

func TestChatLoadHistory(t *testing.T) {
	m := NewChatModel(testStyles())
	msgs := []core.Message{
		{Role: core.RoleUser, Content: "question"},
		{Role: core.RoleAssistant, Content: "answer"},
		{Role: core.RoleSystem, Content: "system prompt"}, // should be skipped
		{Role: core.RoleUser, Content: "follow-up"},
	}
	m.LoadHistory(msgs)

	if len(m.Messages()) != 3 {
		t.Fatalf("LoadHistory produced %d messages, want 3 (system skipped)", len(m.Messages()))
	}
	if m.Messages()[0].Role != "user" || m.Messages()[0].Content != "question" {
		t.Errorf("message[0] = %+v, want user 'question'", m.Messages()[0])
	}
	if m.Messages()[1].Role != "assistant" || m.Messages()[1].Content != "answer" {
		t.Errorf("message[1] = %+v, want assistant 'answer'", m.Messages()[1])
	}
	if m.Messages()[2].Role != "user" || m.Messages()[2].Content != "follow-up" {
		t.Errorf("message[2] = %+v, want user 'follow-up'", m.Messages()[2])
	}

	// Verify it renders
	got := m.Render()
	if !strings.Contains(got, "question") || !strings.Contains(got, "answer") {
		t.Error("rendered output should contain loaded history messages")
	}
}

func TestChatLoadHistoryClearsPrevious(t *testing.T) {
	m := NewChatModel(testStyles())
	m.AppendMessage("user", "old message", "coder", "gpt-4")

	m.LoadHistory([]core.Message{
		{Role: core.RoleUser, Content: "new question"},
	})

	if len(m.Messages()) != 1 {
		t.Fatalf("LoadHistory should clear previous messages, got %d", len(m.Messages()))
	}
	if m.Messages()[0].Content != "new question" {
		t.Errorf("message[0].Content = %q, want %q", m.Messages()[0].Content, "new question")
	}
}
