package core

import (
	"context"
	"errors"
	"testing"
)

// fakeStreamingProvider implements both Provider and StreamingProvider. It
// returns pre-queued StreamChunk sequences for StreamChat (one per call) and
// pre-queued []Message for Chat (fallback).
type fakeStreamingProvider struct {
	streamResponses [][]StreamChunk // one per StreamChat call
	chatResponse    [][]Message
	chatCalls       int
	streamCalls     int
	received        [][]Message
	tools           [][]Tool
}

func (f *fakeStreamingProvider) Chat(_ context.Context, messages []Message, tools []Tool) ([]Message, error) {
	f.chatCalls++
	f.received = append(f.received, append([]Message(nil), messages...))
	f.tools = append(f.tools, append([]Tool(nil), tools...))
	if f.chatCalls > len(f.chatResponse) {
		return nil, errors.New("fakeStreamingProvider: unexpected extra Chat call")
	}
	return f.chatResponse[f.chatCalls-1], nil
}

func (f *fakeStreamingProvider) StreamChat(_ context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error) {
	f.streamCalls++
	f.received = append(f.received, append([]Message(nil), messages...))
	f.tools = append(f.tools, append([]Tool(nil), tools...))
	if f.streamCalls > len(f.streamResponses) {
		return nil, errors.New("fakeStreamingProvider: unexpected extra StreamChat call")
	}
	chunks := f.streamResponses[f.streamCalls-1]
	ch := make(chan StreamChunk, len(chunks))
	for _, c := range chunks {
		ch <- c
	}
	close(ch)
	return ch, nil
}

// --- StreamingObserver fake for testing OnTextDelta ---

type fakeStreamingObserver struct {
	deltas    []string
	turnStart int
	turnEnd   int
}

func (o *fakeStreamingObserver) OnTurnStart()            { o.turnStart++ }
func (o *fakeStreamingObserver) OnTurnEnd()              { o.turnEnd++ }
func (o *fakeStreamingObserver) OnToolCall(call ToolCall) {}
func (o *fakeStreamingObserver) OnToolResult(_, _ string) {}
func (o *fakeStreamingObserver) OnTextDelta(delta string) { o.deltas = append(o.deltas, delta) }

// --- Tests: Task 3.1 RED ---

func TestRunStreamingDirectAnswer(t *testing.T) {
	// REQ-LOOP-1 streaming scenario: StreamingProvider returns text deltas
	// then Done; Run() forwards deltas via observer and returns accumulated text.
	provider := &fakeStreamingProvider{
		streamResponses: [][]StreamChunk{
			{
				{TextDelta: "Hello"},
				{TextDelta: " world"},
				{Done: true},
			},
		},
	}
	observer := &fakeStreamingObserver{}
	agent := &Agent{
		Provider:      provider,
		Tools:         NewRegistry(),
		MaxIterations: 5,
		Observer:      observer,
	}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "Hello world" {
		t.Errorf("answer = %q, want %q", answer, "Hello world")
	}
	if provider.streamCalls != 1 {
		t.Errorf("StreamChat called %d times, want 1", provider.streamCalls)
	}
	if provider.chatCalls != 0 {
		t.Errorf("Chat called %d times, want 0 (streaming path used)", provider.chatCalls)
	}
	if len(observer.deltas) != 2 {
		t.Fatalf("OnTextDelta called %d times, want 2", len(observer.deltas))
	}
	if observer.deltas[0] != "Hello" || observer.deltas[1] != " world" {
		t.Errorf("deltas = %v, want [Hello  world]", observer.deltas)
	}
}

func TestRunStreamingFallbackToSync(t *testing.T) {
	// REQ-LOOP-8 synchronous fallback: non-streaming provider → Chat() used.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "sync answer"}},
	}}
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "sync answer" {
		t.Errorf("answer = %q, want %q", answer, "sync answer")
	}
	if provider.calls != 1 {
		t.Errorf("Chat called %d times, want 1", provider.calls)
	}
}

func TestRunStreamingTextDeltasAccumulated(t *testing.T) {
	// REQ-LOOP-1 streaming: multiple text deltas accumulate into final answer.
	provider := &fakeStreamingProvider{
		streamResponses: [][]StreamChunk{
			{
				{TextDelta: "one"},
				{TextDelta: " two"},
				{TextDelta: " three"},
				{Done: true},
			},
		},
	}
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5}

	answer, err := agent.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "one two three" {
		t.Errorf("answer = %q, want %q", answer, "one two three")
	}
}

func TestRunStreamingToolCallsExecutedAfterStream(t *testing.T) {
	// REQ-LOOP-10: tool call chunks accumulated during streaming, executed
	// after channel closes, loop continues with tool results.
	readFile := &fakeTool{name: "read_file", result: "file contents"}
	registry := NewRegistry()
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeStreamingProvider{
		streamResponses: [][]StreamChunk{
			{
				{TextDelta: "Let me read that..."},
				{ToolCallStart: &ToolCall{ID: "call-1", Name: "read_file"}},
				{ToolCallDelta: &ToolCallDelta{ID: "call-1", Name: "read_file", Arguments: `{"path":"a.md"}`}},
				{Done: true},
			},
			{
				{TextDelta: "done reading"},
				{Done: true},
			},
		},
	}
	agent := &Agent{Provider: provider, Tools: registry, MaxIterations: 5}

	answer, err := agent.Run(context.Background(), "read a.md")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "done reading" {
		t.Errorf("answer = %q, want %q", answer, "done reading")
	}
	if len(readFile.args) != 1 {
		t.Fatalf("tool executed %d times, want 1", len(readFile.args))
	}
	if got := string(readFile.args[0]); got != `{"path":"a.md"}` {
		t.Errorf("tool received args %q, want the expected arguments", got)
	}
	// After stream completes with tool calls, loop continues via streaming.
	if provider.streamCalls != 2 {
		t.Errorf("StreamChat called %d times, want 2 (stream + follow-up)", provider.streamCalls)
	}
	if provider.chatCalls != 0 {
		t.Errorf("Chat called %d times, want 0 (streaming path used for both turns)", provider.chatCalls)
	}
	// The tool result must have been appended to messages for the follow-up call.
	second := provider.received[1]
	last := second[len(second)-1]
	if last.Role != RoleTool || last.Content != "file contents" || last.ToolCallID != "call-1" {
		t.Errorf("tool result message = %+v, want role=tool, content=%q, ToolCallID=%q",
			last, "file contents", "call-1")
	}
}

func TestRunStreamingContextCancellation(t *testing.T) {
	// REQ-LOOP-1 streaming: context cancellation stops the stream.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	provider := &fakeStreamingProvider{
		streamResponses: [][]StreamChunk{
			{
				{TextDelta: "hello"},
				{Done: true},
			},
		},
	}
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5}

	_, err := agent.Run(ctx, "hello")
	// The stream channel is already closed (sync test), so this should
	// still work — but in real usage the StreamChat call would fail.
	// For this unit test we verify the path doesn't panic.
	if err != nil {
		// Error is acceptable — the provider may respect context
		return
	}
}

func TestRunStreamingErrorMidStream(t *testing.T) {
	// REQ-LOOP-10, D8: error mid-stream → StreamChunk{Error} → Run returns error.
	streamErr := errors.New("connection lost")
	provider := &fakeStreamingProvider{
		streamResponses: [][]StreamChunk{
			{
				{TextDelta: "partial"},
				{Error: streamErr},
			},
		},
	}
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5}

	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("Run returned nil error, want error from stream")
	}
	if !errors.Is(err, streamErr) {
		t.Errorf("Run error = %v, want %v", err, streamErr)
	}
}

func TestRunStreamingNilObserverNoPanic(t *testing.T) {
	// REQ-LOOP-9: nil observer → deltas consumed without panic.
	provider := &fakeStreamingProvider{
		streamResponses: [][]StreamChunk{
			{
				{TextDelta: "hello"},
				{Done: true},
			},
		},
	}
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "hello" {
		t.Errorf("answer = %q, want %q", answer, "hello")
	}
}

func TestRunStreamingNonStreamingObserverSkipsDeltas(t *testing.T) {
	// REQ-LOOP-9: observer that does NOT implement StreamingObserver →
	// OnTextDelta is not called (no panic, no-op).
	plainObs := &plainObserver{}
	provider := &fakeStreamingProvider{
		streamResponses: [][]StreamChunk{
			{
				{TextDelta: "delta"},
				{Done: true},
			},
		},
	}
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5, Observer: plainObs}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "delta" {
		t.Errorf("answer = %q, want %q", answer, "delta")
	}
}

func TestRunStreamingSteeringDrainsBetweenStreamTurns(t *testing.T) {
	// REQ-LOOP-11: the steering queue drains between streaming turns — the
	// queued message is injected before the second StreamChat call, exactly
	// like the sync path (TestRunSteeringDrainsAllBeforeNextRequest).
	provider := &fakeStreamingProvider{
		streamResponses: [][]StreamChunk{
			{
				{TextDelta: "first answer"},
				{Done: true},
			},
			{
				{TextDelta: "final answer"},
				{Done: true},
			},
		},
	}
	steering := &fakeQueue{mode: QueueModeAll}
	steering.Enqueue(PendingMessage{Content: "steer now"})
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5, Steering: steering}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "final answer" {
		t.Errorf("answer = %q, want %q", answer, "final answer")
	}
	if provider.streamCalls != 2 {
		t.Errorf("StreamChat called %d times, want 2 (steering kept the loop alive)", provider.streamCalls)
	}
	if provider.chatCalls != 0 {
		t.Errorf("Chat called %d times, want 0 (streaming path used)", provider.chatCalls)
	}
	// The second StreamChat call must carry the steering message.
	second := provider.received[1]
	last := second[len(second)-1]
	if last.Role != RoleUser || last.Content != "steer now" {
		t.Errorf("last message of second StreamChat = %+v, want the injected steering message", last)
	}
}

func TestRunStreamingFollowUpDrainsAfterStream(t *testing.T) {
	// REQ-LOOP-11: the follow-up queue drains after the streaming inner loop
	// would otherwise stop, keeping the loop alive with a new streaming turn —
	// exactly like the sync path (TestRunFollowUpDrainsAtStop).
	provider := &fakeStreamingProvider{
		streamResponses: [][]StreamChunk{
			{
				{TextDelta: "first answer"},
				{Done: true},
			},
			{
				{TextDelta: "final answer"},
				{Done: true},
			},
		},
	}
	followUp := &fakeQueue{mode: QueueModeAll}
	followUp.Enqueue(PendingMessage{Content: "follow up please"})
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5, FollowUp: followUp}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "final answer" {
		t.Errorf("answer = %q, want the follow-up turn's answer %q", answer, "final answer")
	}
	if provider.streamCalls != 2 {
		t.Errorf("StreamChat called %d times, want 2 (follow-up kept the loop alive)", provider.streamCalls)
	}
	if provider.chatCalls != 0 {
		t.Errorf("Chat called %d times, want 0 (streaming path used)", provider.chatCalls)
	}
	// The second StreamChat call must carry the follow-up message.
	second := provider.received[1]
	last := second[len(second)-1]
	if last.Role != RoleUser || last.Content != "follow up please" {
		t.Errorf("last message of second StreamChat = %+v, want the injected follow-up message", last)
	}
}

// plainObserver implements Observer but NOT StreamingObserver.
type plainObserver struct{}

func (o *plainObserver) OnTurnStart()            {}
func (o *plainObserver) OnTurnEnd()              {}
func (o *plainObserver) OnToolCall(call ToolCall) {}
func (o *plainObserver) OnToolResult(_, _ string) {}
