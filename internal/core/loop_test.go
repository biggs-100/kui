package core

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// fakeProvider returns pre-queued responses, one per Chat call, and records
// the message history it receives so tests can assert tool results are fed
// back to the provider (REQ-LOOP-4).
type fakeProvider struct {
	responses [][]Message
	calls     int
	received  [][]Message
}

func (f *fakeProvider) Chat(_ context.Context, messages []Message, _ []Tool) ([]Message, error) {
	f.calls++
	f.received = append(f.received, append([]Message(nil), messages...))
	if f.calls > len(f.responses) {
		return nil, errors.New("fakeProvider: unexpected extra Chat call")
	}
	return f.responses[f.calls-1], nil
}

// fakeTool is an in-memory Tool implementation that records the raw JSON
// arguments it received and returns a fixed result or error.
type fakeTool struct {
	name   string
	result string
	err    error
	args   []json.RawMessage
}

func (t *fakeTool) Name() string        { return t.name }
func (t *fakeTool) Description() string { return "fake tool: " + t.name }
func (t *fakeTool) Schema() string      { return `{"type":"object","properties":{}}` }

func (t *fakeTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	t.args = append(t.args, append(json.RawMessage(nil), args...))
	return t.result, t.err
}

func toolCall(id, name, arguments string) Message {
	return Message{Role: RoleAssistant, ToolCall: &ToolCall{ID: id, Name: name, Arguments: arguments}}
}

func TestRunDirectAnswerWithoutTools(t *testing.T) {
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "hello there"}},
	}}
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "hello there" {
		t.Errorf("answer = %q, want %q", answer, "hello there")
	}
	if provider.calls != 1 {
		t.Errorf("provider called %d times, want 1", provider.calls)
	}
	first := provider.received[0]
	if len(first) != 1 || first[0].Role != RoleUser || first[0].Content != "hello" {
		t.Errorf("first Chat received %+v, want single user message with prompt", first)
	}
}

func TestRunMultiStepToolResolution(t *testing.T) {
	readFile := &fakeTool{name: "read_file", result: "file contents"}
	registry := NewRegistry()
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "read_file", `{"path":"notes.md"}`)},
		{{Role: RoleAssistant, Content: "done reading"}},
	}}
	agent := &Agent{Provider: provider, Tools: registry, MaxIterations: 5}

	answer, err := agent.Run(context.Background(), "read notes.md")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "done reading" {
		t.Errorf("answer = %q, want %q", answer, "done reading")
	}
	if len(readFile.args) != 1 {
		t.Fatalf("tool executed %d times, want 1", len(readFile.args))
	}
	if got := string(readFile.args[0]); got != `{"path":"notes.md"}` {
		t.Errorf("tool received args %q, want the requested arguments", got)
	}

	// The provider must have been called again with the tool result and its
	// tool-call identifier (REQ-LOOP-4).
	if provider.calls != 2 {
		t.Fatalf("provider called %d times, want 2", provider.calls)
	}
	second := provider.received[1]
	if len(second) < 2 {
		t.Fatalf("second Chat received %d messages, want tool result appended", len(second))
	}
	last := second[len(second)-1]
	if last.Role != RoleTool || last.Content != "file contents" || last.ToolCallID != "call-1" {
		t.Errorf("tool result message = %+v, want role=tool, content=%q, ToolCallID=%q",
			last, "file contents", "call-1")
	}
}

func TestRunUnknownToolTerminatesWithTypedError(t *testing.T) {
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "no_such_tool", `{}`)},
	}}
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5}

	_, err := agent.Run(context.Background(), "do the thing")
	var unknown *UnknownToolError
	if !errors.As(err, &unknown) {
		t.Fatalf("Run error = %v, want *UnknownToolError", err)
	}
	if unknown.Name != "no_such_tool" {
		t.Errorf("UnknownToolError.Name = %q, want %q", unknown.Name, "no_such_tool")
	}
	if provider.calls != 1 {
		t.Errorf("provider called %d times, want 1 (no further requests after unknown tool)", provider.calls)
	}
}

func TestRunIterationBudgetExhausted(t *testing.T) {
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("c1", "read_file", `{}`)},
		{toolCall("c2", "read_file", `{}`)},
		{toolCall("c3", "read_file", `{}`)},
	}}
	readFile := &fakeTool{name: "read_file", result: "ok"}
	registry := NewRegistry()
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	agent := &Agent{Provider: provider, Tools: registry, MaxIterations: 3}

	_, err := agent.Run(context.Background(), "loop forever")
	var limit *IterationLimitError
	if !errors.As(err, &limit) {
		t.Fatalf("Run error = %v, want *IterationLimitError", err)
	}
	if limit.Max != 3 {
		t.Errorf("IterationLimitError.Max = %d, want 3", limit.Max)
	}
	if provider.calls != 3 {
		t.Errorf("provider called %d times, want exactly 3 (budget)", provider.calls)
	}
}

func TestRunToolFailureNamesFailingTool(t *testing.T) {
	cause := errors.New("permission denied")
	boom := &fakeTool{name: "write_file", err: cause}
	registry := NewRegistry()
	if err := registry.Register(boom); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-9", "write_file", `{"path":"x"}`)},
	}}
	agent := &Agent{Provider: provider, Tools: registry, MaxIterations: 5}

	_, err := agent.Run(context.Background(), "write x")
	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		t.Fatalf("Run error = %v, want *ToolError", err)
	}
	if toolErr.Name != "write_file" {
		t.Errorf("ToolError.Name = %q, want %q", toolErr.Name, "write_file")
	}
	if !errors.Is(toolErr, cause) {
		t.Errorf("ToolError does not unwrap to cause %v (got %v)", cause, toolErr.Err)
	}
}

func TestRegistryPreservesRegistrationOrder(t *testing.T) {
	registry := NewRegistry()
	for _, tool := range []*fakeTool{
		{name: "bash"},
		{name: "read_file"},
		{name: "write_file"},
	} {
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Register(%s) failed: %v", tool.name, err)
		}
	}

	listed := registry.List()
	if len(listed) != 3 {
		t.Fatalf("List() returned %d tools, want 3", len(listed))
	}
	var names []string
	for _, tool := range listed {
		names = append(names, tool.Name())
	}
	want := []string{"bash", "read_file", "write_file"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("registration order = %v, want %v", names, want)
	}
}

func TestRegistryGetAndDuplicateRegistration(t *testing.T) {
	registry := NewRegistry()
	readFile := &fakeTool{name: "read_file"}
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if got, ok := registry.Get("read_file"); !ok || got != readFile {
		t.Errorf("Get(read_file) = %v, %v; want the registered tool, true", got, ok)
	}
	if _, ok := registry.Get("missing"); ok {
		t.Error("Get(missing) reported ok, want not ok")
	}
	if err := registry.Register(&fakeTool{name: "read_file"}); err == nil {
		t.Error("duplicate Register returned nil error, want error")
	}
}
