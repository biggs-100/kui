package tui

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/biggs-100/kui/internal/core"
)

type fakeTelemetryTool struct{}

func (fakeTelemetryTool) Name() string        { return "fake" }
func (fakeTelemetryTool) Description() string { return "fake tool" }
func (fakeTelemetryTool) Schema() string      { return "{}" }
func (fakeTelemetryTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return "ok", nil
}

// TestSpyToolEmitsRealEvents proves the TUI sees REAL tool activity: a
// wrapped tool emits a call event before executing and a result event after.
func TestSpyToolEmitsRealEvents(t *testing.T) {
	c := NewController([]string{"coder"}, nil, nil)
	hub := &toolHub{}
	hub.bind(c)

	tool := hub.wrap(fakeTelemetryTool{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := tool.Execute(context.Background(), json.RawMessage(`{}`)); err != nil {
			t.Errorf("execute failed: %v", err)
		}
	}()

	select {
	case ev := <-c.Events():
		call, ok := ev.(toolCallMsg)
		if !ok || call.name != "fake" || call.callID == "" {
			t.Fatalf("expected toolCallMsg for fake, got %#v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for tool call event")
	}

	select {
	case ev := <-c.Events():
		res, ok := ev.(toolResultMsg)
		if !ok || res.callID == "" || !strings.Contains(res.result, "ok") {
			t.Fatalf("expected toolResultMsg carrying result, got %#v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for tool result event")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("execute did not finish")
	}
}

var _ = core.Tool(nil)
