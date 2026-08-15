package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// fakeProvider returns pre-queued responses, one per Chat call, and records
// the message history and the advertised tool slice it receives so tests can
// assert tool results are fed back (REQ-LOOP-4) and denied tools are filtered
// out before the provider request (D15, REQ-PERM-3).
type fakeProvider struct {
	responses [][]Message
	calls     int
	received  [][]Message
	tools     [][]Tool
}

func (f *fakeProvider) Chat(_ context.Context, messages []Message, tools []Tool) ([]Message, error) {
	f.calls++
	f.received = append(f.received, append([]Message(nil), messages...))
	f.tools = append(f.tools, append([]Tool(nil), tools...))
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

// fakeQueue is an in-memory PendingQueue fake honouring QueueMode (D19). The
// concrete mutex queue lives in internal/agent (PR 4); core only consumes the
// port. It records every drain so tests can prove failed turns never drain.
type fakeQueue struct {
	mode       QueueMode
	queue      []PendingMessage
	drainCount int
}

func (q *fakeQueue) Enqueue(messages ...PendingMessage) {
	q.queue = append(q.queue, messages...)
}

func (q *fakeQueue) Drain() []PendingMessage {
	q.drainCount++
	if len(q.queue) == 0 {
		return nil
	}
	if q.mode == QueueModeOneAtATime {
		head := q.queue[0]
		q.queue = q.queue[1:]
		return []PendingMessage{head}
	}
	drained := q.queue
	q.queue = nil
	return drained
}

// fakeProfileManager is a ProfileManager fake that records every switch
// request it receives and returns pre-queued messages or an error, so loop
// tests can prove switch application between turns (REQ-LOOP-5/6, D16). The
// zero value (no messages, no error) is a no-op, keeping the nil-safe port
// test behavior identical to the single-level loop (D14).
type fakeProfileManager struct {
	names    []string
	messages []Message
	err      error
}

func (f *fakeProfileManager) ApplySwitch(_ context.Context, name string) ([]Message, error) {
	f.names = append(f.names, name)
	return f.messages, f.err
}

// fakePermissionEvaluator gates on a deny set: Allow reports false for denied
// names and Filter drops them from the advertised slice. An empty deny set is
// fully permissive, so the nil-safe port test still behaves like the
// single-level loop (D14).
type fakePermissionEvaluator struct {
	deny map[string]bool
}

func (f *fakePermissionEvaluator) Allow(name string) bool { return !f.deny[name] }

func (f *fakePermissionEvaluator) Filter(tools []Tool) []Tool {
	filtered := make([]Tool, 0, len(tools))
	for _, tool := range tools {
		if !f.deny[tool.Name()] {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

func TestRunSteeringDrainsAllBeforeNextRequest(t *testing.T) {
	// REQ-QUEUE-1, drain-all: three queued steering messages are all injected
	// before the next provider request, in order.
	readFile := &fakeTool{name: "read_file", result: "contents"}
	registry := NewRegistry()
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "read_file", `{"path":"a"}`)},
		{{Role: RoleAssistant, Content: "done"}},
	}}
	steering := &fakeQueue{mode: QueueModeAll}
	steering.Enqueue(
		PendingMessage{Content: "steer one"},
		PendingMessage{Content: "steer two"},
		PendingMessage{Content: "steer three"},
	)
	agent := &Agent{Provider: provider, Tools: registry, MaxIterations: 5, Steering: steering}

	answer, err := agent.Run(context.Background(), "start")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "done" {
		t.Errorf("answer = %q, want %q", answer, "done")
	}
	if provider.calls != 2 {
		t.Fatalf("provider called %d times, want 2", provider.calls)
	}
	second := provider.received[1]
	if len(second) < 3 {
		t.Fatalf("second Chat received %d messages, want the 3 queued messages injected", len(second))
	}
	got := []string{
		second[len(second)-3].Content,
		second[len(second)-2].Content,
		second[len(second)-1].Content,
	}
	want := []string{"steer one", "steer two", "steer three"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("queued messages injected = %v, want %v", got, want)
	}
}

func TestRunSteeringDrainsOnePerTurn(t *testing.T) {
	// REQ-QUEUE-1, one-at-a-time: exactly one queued message is injected per
	// turn, in FIFO order.
	readFile := &fakeTool{name: "read_file", result: "ok"}
	registry := NewRegistry()
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("c1", "read_file", `{}`)},
		{toolCall("c2", "read_file", `{}`)},
		{toolCall("c3", "read_file", `{}`)},
		{{Role: RoleAssistant, Content: "finished"}},
	}}
	steering := &fakeQueue{mode: QueueModeOneAtATime}
	for i := 1; i <= 3; i++ {
		steering.Enqueue(PendingMessage{Content: fmt.Sprintf("steer %d", i)})
	}
	agent := &Agent{Provider: provider, Tools: registry, MaxIterations: 5, Steering: steering}

	answer, err := agent.Run(context.Background(), "start")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "finished" {
		t.Errorf("answer = %q, want %q", answer, "finished")
	}
	// Each provider call after the first carries exactly one new queued
	// message: steer 1 on call 2, steer 2 on call 3, steer 3 on call 4.
	for i, call := range []int{2, 3, 4} {
		received := provider.received[call-1]
		last := received[len(received)-1]
		want := fmt.Sprintf("steer %d", i+1)
		if last.Role != RoleUser || last.Content != want {
			t.Errorf("call %d last message = %+v, want user message %q", call, last, want)
		}
	}
	if provider.calls != 4 {
		t.Errorf("provider called %d times, want 4", provider.calls)
	}
}

func TestRunFollowUpDrainsAtStop(t *testing.T) {
	// REQ-QUEUE-2: when the provider returns without tool calls and the
	// follow-up queue holds a message, the message is injected as a new turn
	// and the loop continues instead of stopping.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "first answer"}},
		{{Role: RoleAssistant, Content: "final answer"}},
	}}
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
	if provider.calls != 2 {
		t.Fatalf("provider called %d times, want 2 (follow-up kept the loop alive)", provider.calls)
	}
	second := provider.received[1]
	last := second[len(second)-1]
	if last.Role != RoleUser || last.Content != "follow up please" {
		t.Errorf("last message of second Chat = %+v, want the injected follow-up message", last)
	}
}

func TestRunFollowUpEmptyStopsNormally(t *testing.T) {
	// REQ-QUEUE-2, empty queue: with an empty follow-up queue the loop stops
	// normally on the first answer, exactly like the single-level loop.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "hello there"}},
	}}
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		FollowUp: &fakeQueue{mode: QueueModeAll},
	}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "hello there" {
		t.Errorf("answer = %q, want %q", answer, "hello there")
	}
	if provider.calls != 1 {
		t.Errorf("provider called %d times, want 1 (empty follow-up stops the loop)", provider.calls)
	}
}

func TestRunFollowUpWaitsForEmptySteering(t *testing.T) {
	// REQ-QUEUE-2: the follow-up queue drains only when the steering queue is
	// empty; a non-empty steering queue is injected first and keeps the loop
	// alive without touching the follow-up queue.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "answer one"}},
		{{Role: RoleAssistant, Content: "answer two"}},
		{{Role: RoleAssistant, Content: "answer three"}},
	}}
	steering := &fakeQueue{mode: QueueModeAll}
	steering.Enqueue(PendingMessage{Content: "steer first"})
	followUp := &fakeQueue{mode: QueueModeAll}
	followUp.Enqueue(PendingMessage{Content: "follow last"})
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		Steering: steering, FollowUp: followUp,
	}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "answer three" {
		t.Errorf("answer = %q, want %q", answer, "answer three")
	}
	// Call 2 ends with the steering message; call 3 ends with the follow-up.
	second := provider.received[1]
	if last := second[len(second)-1]; last.Content != "steer first" {
		t.Errorf("second Chat last message = %+v, want the steering message", last)
	}
	third := provider.received[2]
	if last := third[len(third)-1]; last.Content != "follow last" {
		t.Errorf("third Chat last message = %+v, want the follow-up message", last)
	}
	if provider.calls != 3 {
		t.Errorf("provider called %d times, want 3", provider.calls)
	}
}

func TestRunBudgetCountsFollowUpContinuations(t *testing.T) {
	// REQ-QUEUE-3: follow-up continuations count toward the iteration
	// budget; exhaustion returns the iteration-limit error and no further
	// provider requests are made.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "a"}},
		{{Role: RoleAssistant, Content: "b"}},
		{{Role: RoleAssistant, Content: "c"}},
	}}
	// One-at-a-time mode keeps the loop alive for exactly one continuation
	// per drain, so the follow-up queue drives the loop into the budget.
	followUp := &fakeQueue{mode: QueueModeOneAtATime}
	followUp.Enqueue(
		PendingMessage{Content: "again"},
		PendingMessage{Content: "again"},
		PendingMessage{Content: "again"},
	)
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 3, FollowUp: followUp}

	_, err := agent.Run(context.Background(), "loop")
	var limit *IterationLimitError
	if !errors.As(err, &limit) {
		t.Fatalf("Run error = %v, want *IterationLimitError", err)
	}
	if limit.Max != 3 {
		t.Errorf("IterationLimitError.Max = %d, want 3", limit.Max)
	}
	if provider.calls != 3 {
		t.Errorf("provider called %d times, want exactly 3 (budget counts follow-ups)", provider.calls)
	}
}

func TestRunToolFailureSkipsQueuedSteering(t *testing.T) {
	// REQ-QUEUE-3: a failing tool terminates the loop with the tool's error
	// and the queued steering message is never injected — the failed turn
	// never drains, so no partial state reaches the conversation.
	cause := errors.New("boom")
	boom := &fakeTool{name: "write_file", err: cause}
	registry := NewRegistry()
	if err := registry.Register(boom); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "write_file", `{"path":"x"}`)},
	}}
	steering := &fakeQueue{mode: QueueModeAll}
	steering.Enqueue(PendingMessage{Content: "must not be injected"})
	agent := &Agent{Provider: provider, Tools: registry, MaxIterations: 5, Steering: steering}

	_, err := agent.Run(context.Background(), "write x")
	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		t.Fatalf("Run error = %v, want *ToolError", err)
	}
	if provider.calls != 1 {
		t.Errorf("provider called %d times, want 1 (loop stops at the failing tool)", provider.calls)
	}
	for _, m := range provider.received[0] {
		if m.Content == "must not be injected" {
			t.Errorf("queued message leaked into the conversation: %+v", m)
		}
	}
	if steering.drainCount != 0 {
		t.Errorf("steering drained %d times, want 0 (a failed turn never drains)", steering.drainCount)
	}
	if len(steering.queue) != 1 {
		t.Errorf("steering queue holds %d messages, want 1 (message must stay queued)", len(steering.queue))
	}
}

func TestRunNilSafeWhenPortsUnset(t *testing.T) {
	// D14: with all four new ports set to non-nil fakes and empty queues,
	// behavior is identical to the single-level loop.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "plain answer"}},
	}}
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		Steering:    &fakeQueue{mode: QueueModeAll},
		FollowUp:    &fakeQueue{mode: QueueModeAll},
		Profiles:    &fakeProfileManager{},
		Permissions: &fakePermissionEvaluator{},
	}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "plain answer" {
		t.Errorf("answer = %q, want %q", answer, "plain answer")
	}
	if provider.calls != 1 {
		t.Errorf("provider called %d times, want 1", provider.calls)
	}
}

func TestRunFiltersDeniedToolsBeforeChat(t *testing.T) {
	// D15, REQ-PERM-3: the tools slice handed to Chat must exclude denied
	// tools while keeping allowed ones, so the provider never advertises
	// them. Denied tools must also never execute.
	readFile := &fakeTool{name: "read_file", result: "contents"}
	writeFile := &fakeTool{name: "write_file", result: "written"}
	registry := NewRegistry()
	for _, tool := range []*fakeTool{readFile, writeFile} {
		if err := registry.Register(tool); err != nil {
			t.Fatalf("Register(%s) failed: %v", tool.name, err)
		}
	}
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "done"}},
	}}
	permissions := &fakePermissionEvaluator{deny: map[string]bool{"write_file": true}}
	agent := &Agent{Provider: provider, Tools: registry, MaxIterations: 5, Permissions: permissions}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "done" {
		t.Errorf("answer = %q, want %q", answer, "done")
	}
	if provider.calls != 1 {
		t.Fatalf("provider called %d times, want 1", provider.calls)
	}
	var got []string
	for _, tool := range provider.tools[0] {
		got = append(got, tool.Name())
	}
	if want := []string{"read_file"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tools advertised to Chat = %v, want %v (write_file must be hidden)", got, want)
	}
	if len(writeFile.args) != 0 {
		t.Errorf("denied tool write_file executed %d times, want 0", len(writeFile.args))
	}
}

func TestRunDeniedDispatchReturnsPermissionError(t *testing.T) {
	// REQ-PERM-4, defense in depth: even if the provider requests a denied
	// tool (it should never see one), the loop rejects the dispatch with a
	// typed PermissionError and the tool's side effect never runs.
	readSecrets := &fakeTool{name: "read_secrets", result: "top secret"}
	registry := NewRegistry()
	if err := registry.Register(readSecrets); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "read_secrets", `{}`)},
	}}
	permissions := &fakePermissionEvaluator{deny: map[string]bool{"read_secrets": true}}
	agent := &Agent{Provider: provider, Tools: registry, MaxIterations: 5, Permissions: permissions}

	_, err := agent.Run(context.Background(), "read the secrets")
	var permErr *PermissionError
	if !errors.As(err, &permErr) {
		t.Fatalf("Run error = %v, want *PermissionError", err)
	}
	if permErr.Tool != "read_secrets" {
		t.Errorf("PermissionError.Tool = %q, want %q", permErr.Tool, "read_secrets")
	}
	if len(readSecrets.args) != 0 {
		t.Errorf("denied tool executed %d times, want 0 (no side effect)", len(readSecrets.args))
	}
	if provider.calls != 1 {
		t.Errorf("provider called %d times, want 1 (no further requests after denial)", provider.calls)
	}
}

func TestRunSwitchAppliesBetweenTurns(t *testing.T) {
	// REQ-LOOP-5: a switch queued during a tool call applies before the next
	// provider request — never mid-tool-call. The switch messages (new system
	// prompt + profile-context marker, REQ-LOOP-6) are appended to the
	// history, which itself is preserved unchanged (REQ-PROFILE-3).
	readFile := &fakeTool{name: "read_file", result: "contents"}
	registry := NewRegistry()
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "read_file", `{"path":"a"}`)},
		{{Role: RoleAssistant, Content: "done under coder"}},
	}}
	profiles := &fakeProfileManager{
		messages: []Message{
			{Role: RoleSystem, Content: "you are the coder profile"},
			{Role: RoleSystem, Content: "Profile switched to coder. Continue with the existing conversation context..."},
		},
	}
	steering := &fakeQueue{mode: QueueModeAll}
	steering.Enqueue(PendingMessage{SwitchProfile: "coder"})
	agent := &Agent{
		Provider: provider, Tools: registry, MaxIterations: 5,
		Steering: steering, Profiles: profiles,
	}

	answer, err := agent.Run(context.Background(), "read a")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "done under coder" {
		t.Errorf("answer = %q, want %q", answer, "done under coder")
	}
	if len(profiles.names) != 1 || profiles.names[0] != "coder" {
		t.Errorf("ApplySwitch called with %v, want exactly [coder]", profiles.names)
	}
	if provider.calls != 2 {
		t.Fatalf("provider called %d times, want 2 (switch kept the loop alive)", provider.calls)
	}

	second := provider.received[1]
	// History is preserved: prompt, tool call, and tool result are all still
	// present before the switch messages.
	if len(second) != 5 {
		t.Fatalf("second Chat received %d messages, want 5 (3 history + system prompt + marker)", len(second))
	}
	if last := second[2]; last.Role != RoleTool || last.ToolCallID != "call-1" {
		t.Errorf("history lost the tool result: %+v", last)
	}
	sys := second[3]
	if sys.Role != RoleSystem || sys.Content != "you are the coder profile" {
		t.Errorf("switch system prompt message = %+v, want the coder system prompt", sys)
	}
	marker := second[4]
	if marker.Role != RoleSystem || !strings.Contains(marker.Content, "coder") {
		t.Errorf("marker message = %+v, want a RoleSystem message naming coder", marker)
	}
}

func TestRunUnknownProfileReturnsTypedError(t *testing.T) {
	// REQ-PROFILE-3: a switch naming an unknown profile aborts the run with a
	// typed UnknownProfileError and no further provider requests are made.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "first answer"}},
	}}
	profiles := &fakeProfileManager{err: &UnknownProfileError{Name: "nope"}}
	steering := &fakeQueue{mode: QueueModeAll}
	steering.Enqueue(PendingMessage{SwitchProfile: "nope"})
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		Steering: steering, Profiles: profiles,
	}

	_, err := agent.Run(context.Background(), "hello")
	var unknown *UnknownProfileError
	if !errors.As(err, &unknown) {
		t.Fatalf("Run error = %v, want *UnknownProfileError", err)
	}
	if unknown.Name != "nope" {
		t.Errorf("UnknownProfileError.Name = %q, want %q", unknown.Name, "nope")
	}
	if provider.calls != 1 {
		t.Errorf("provider called %d times, want 1 (loop stops at unknown profile)", provider.calls)
	}
}

func TestRunMultipleSwitchesLastWins(t *testing.T) {
	// REQ-LOOP-5: when one steering drain carries two switch requests, the
	// last switch determines the active profile — ApplySwitch is called once
	// with the last name.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "answer one"}},
		{{Role: RoleAssistant, Content: "answer two"}},
	}}
	profiles := &fakeProfileManager{
		messages: []Message{{Role: RoleSystem, Content: "Profile switched to writer. Continue..."}},
	}
	steering := &fakeQueue{mode: QueueModeAll}
	steering.Enqueue(
		PendingMessage{SwitchProfile: "coder"},
		PendingMessage{SwitchProfile: "writer"},
	)
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		Steering: steering, Profiles: profiles,
	}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "answer two" {
		t.Errorf("answer = %q, want %q", answer, "answer two")
	}
	if len(profiles.names) != 1 || profiles.names[0] != "writer" {
		t.Errorf("ApplySwitch called with %v, want exactly [writer] (last switch wins)", profiles.names)
	}
	if provider.calls != 2 {
		t.Errorf("provider called %d times, want 2", provider.calls)
	}
}

func TestRunNoMarkerWithoutSwitch(t *testing.T) {
	// REQ-LOOP-6: a session with no profile switch inserts no marker (or
	// system) messages into the history.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "plain answer"}},
	}}
	agent := &Agent{Provider: provider, Tools: NewRegistry(), MaxIterations: 5}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "plain answer" {
		t.Errorf("answer = %q, want %q", answer, "plain answer")
	}
	for _, m := range provider.received[0] {
		if m.Role == RoleSystem {
			t.Errorf("unexpected system/marker message without a switch: %+v", m)
		}
	}
}
