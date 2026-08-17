package core

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// loopHookSpy records every hook emission with its event name and context
// snapshot so loop tests can assert hook firing and message mutation.
type loopHookSpy struct {
	events  []string
	blocks  []string
	results []string
}

func (s *loopHookSpy) reset() {
	s.events = nil
	s.blocks = nil
	s.results = nil
}

func TestLoopNilHookRegistryUnchanged(t *testing.T) {
	// REQ-LOOP-12, "Nil HookRegistry — backward compatible": a loop with nil
	// HookRegistry must behave identically to a loop without hooks.
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
}

func TestLoopWithHooksBeforeProviderRequestFires(t *testing.T) {
	// REQ-LOOP-12, "Non-nil HookRegistry — hooks fire": a loop with a
	// HookRegistry containing a before_provider_request handler must call it
	// before each LLM request.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "done"}},
	}}
	registry := NewHookRegistry()
	var fired bool
	_ = registry.Register("before_provider_request", func(ctx HookContext) error {
		fired = true
		return nil
	})
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		Hooks: registry,
	}

	_, err := agent.Run(context.Background(), "start")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !fired {
		t.Error("before_provider_request hook was not called")
	}
}

func TestLoopWithHooksBeforeProviderRequestMutatesMessages(t *testing.T) {
	// REQ-LOOP-13: a before_provider_request handler that calls SetMessages
	// must modify the messages sent to the provider.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "done"}},
	}}
	registry := NewHookRegistry()
	_ = registry.Register("before_provider_request", func(ctx HookContext) error {
		msgs := ctx.Messages()
		sys := Message{Role: RoleSystem, Content: "system instruction"}
		ctx.SetMessages(append([]Message{sys}, msgs...))
		return nil
	})
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		Hooks: registry,
	}

	_, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider called %d times, want 1", provider.calls)
	}
	first := provider.received[0]
	if len(first) != 2 {
		t.Fatalf("provider received %d messages, want 2 (system + user)", len(first))
	}
	if first[0].Role != RoleSystem || first[0].Content != "system instruction" {
		t.Errorf("first message = %+v, want system instruction", first[0])
	}
	if first[1].Role != RoleUser || first[1].Content != "hello" {
		t.Errorf("second message = %+v, want user hello", first[1])
	}
}

func TestLoopWithHooksBeforeToolExecutionBlocksTool(t *testing.T) {
	// REQ-LOOP-14: a before_tool_execution handler that calls Block must
	// prevent tool execution and return a blocked-tool result to the provider.
	readFile := &fakeTool{name: "read_file", result: "contents"}
	registry := NewRegistry()
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "read_file", `{"path":"a"}`)},
		{{Role: RoleAssistant, Content: "blocked"}},
	}}
	hookReg := NewHookRegistry()
	_ = hookReg.Register("before_tool_execution", func(ctx HookContext) error {
		if tc := ctx.ToolCall(); tc != nil && tc.Name == "read_file" {
			ctx.Block("policy")
		}
		return nil
	})
	agent := &Agent{
		Provider: provider, Tools: registry, MaxIterations: 5,
		Hooks: hookReg,
	}

	answer, err := agent.Run(context.Background(), "read a")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "blocked" {
		t.Errorf("answer = %q, want %q", answer, "blocked")
	}
	// Tool must not have executed.
	if len(readFile.args) != 0 {
		t.Errorf("tool executed %d times, want 0 (should be blocked)", len(readFile.args))
	}
	// Provider must receive a blocked-tool result message.
	if provider.calls != 2 {
		t.Fatalf("provider called %d times, want 2", provider.calls)
	}
	second := provider.received[1]
	if len(second) < 2 {
		t.Fatalf("second Chat received %d messages, want at least 2", len(second))
	}
	last := second[len(second)-1]
	if last.Role != RoleTool {
		t.Errorf("last message role = %q, want %q", last.Role, RoleTool)
	}
	if last.Content == "" {
		t.Error("blocked-tool result message must have non-empty content")
	}
}

func TestLoopWithHooksAfterToolExecutionObservesResult(t *testing.T) {
	// REQ-LOOP-15: an after_tool_execution handler must see the tool result.
	readFile := &fakeTool{name: "read_file", result: "file contents"}
	registry := NewRegistry()
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "read_file", `{}`)},
		{{Role: RoleAssistant, Content: "done"}},
	}}
	hookReg := NewHookRegistry()
	var observedResult string
	_ = hookReg.Register("after_tool_execution", func(ctx HookContext) error {
		// The after_tool_execution hook sees the tool call; we verify the
		// tool result message was appended by checking the messages slice.
		msgs := ctx.Messages()
		if len(msgs) > 0 {
			last := msgs[len(msgs)-1]
			if last.Role == RoleTool {
				observedResult = last.Content
			}
		}
		return nil
	})
	agent := &Agent{
		Provider: provider, Tools: registry, MaxIterations: 5,
		Hooks: hookReg,
	}

	_, err := agent.Run(context.Background(), "read")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if observedResult != "file contents" {
		t.Errorf("after_tool_execution observed result = %q, want %q", observedResult, "file contents")
	}
}

func TestLoopWithHookErrorDoesNotAbortLoop(t *testing.T) {
	// REQ-LOOP-13, "Hook error does not abort the loop": a before_provider_request
	// handler that returns an error must not abort the loop — the loop continues
	// with the original (unmodified) messages.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "recovered"}},
	}}
	registry := NewHookRegistry()
	_ = registry.Register("before_provider_request", func(ctx HookContext) error {
		return errors.New("hook boom")
	})
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		Hooks: registry,
	}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v, hook error should be logged, not propagated", err)
	}
	if answer != "recovered" {
		t.Errorf("answer = %q, want %q", answer, "recovered")
	}
	if provider.calls != 1 {
		t.Errorf("provider called %d times, want 1", provider.calls)
	}
	// Messages must be unmodified (no system injection from the erroring hook).
	first := provider.received[0]
	if len(first) != 1 || first[0].Role != RoleUser {
		t.Errorf("provider received %+v, want single unmodified user message", first)
	}
}

func TestLoopWithHookPanicDoesNotCrashLoop(t *testing.T) {
	// REQ-LOOP-7, "Observer failure is contained" (same principle for hooks):
	// a panicking hook handler must not crash the loop.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "survived"}},
	}}
	registry := NewHookRegistry()
	_ = registry.Register("before_provider_request", func(ctx HookContext) error {
		panic("hook panic")
	})
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		Hooks: registry,
	}

	answer, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "survived" {
		t.Errorf("answer = %q, want %q", answer, "survived")
	}
}

func TestLoopWithHooksMultipleOnSameEventRunInOrder(t *testing.T) {
	// REQ-HOOK-1: multiple handlers on the same event must run in
	// registration order.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "done"}},
	}}
	registry := NewHookRegistry()
	var order []int
	_ = registry.Register("before_provider_request", func(ctx HookContext) error {
		order = append(order, 1)
		return nil
	})
	_ = registry.Register("before_provider_request", func(ctx HookContext) error {
		order = append(order, 2)
		return nil
	})
	_ = registry.Register("before_provider_request", func(ctx HookContext) error {
		order = append(order, 3)
		return nil
	})
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		Hooks: registry,
	}

	_, err := agent.Run(context.Background(), "start")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(order) != 3 {
		t.Fatalf("hook order = %v, want [1 2 3]", order)
	}
	if order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("hook order = %v, want [1 2 3] (registration order)", order)
	}
}

func TestLoopWithHooksBeforeProviderRequestReplacesSystemMessage(t *testing.T) {
	// REQ-LOOP-13: a before_provider_request handler can replace the entire
	// message set, including replacing a system message.
	provider := &fakeProvider{responses: [][]Message{
		{{Role: RoleAssistant, Content: "done"}},
	}}
	registry := NewHookRegistry()
	_ = registry.Register("before_provider_request", func(ctx HookContext) error {
		ctx.SetMessages([]Message{
			{Role: RoleSystem, Content: "new system prompt"},
			{Role: RoleUser, Content: "modified prompt"},
		})
		return nil
	})
	agent := &Agent{
		Provider: provider, Tools: NewRegistry(), MaxIterations: 5,
		Hooks: registry,
	}

	_, err := agent.Run(context.Background(), "original prompt")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider called %d times, want 1", provider.calls)
	}
	first := provider.received[0]
	if len(first) != 2 {
		t.Fatalf("provider received %d messages, want 2", len(first))
	}
	if first[0].Role != RoleSystem || first[0].Content != "new system prompt" {
		t.Errorf("first message = %+v, want new system prompt", first[0])
	}
	if first[1].Role != RoleUser || first[1].Content != "modified prompt" {
		t.Errorf("second message = %+v, want modified prompt", first[1])
	}
}

func TestLoopWithHooksBeforeToolBlockReturnsBlockedResultToProvider(t *testing.T) {
	// REQ-LOOP-14: when a before_tool_execution handler blocks a tool, the
	// blocked result message must be returned to the provider with a meaningful
	// content string including the reason.
	writeFile := &fakeTool{name: "write_file", result: "written"}
	registry := NewRegistry()
	if err := registry.Register(writeFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "write_file", `{"path":"x"}`)},
		{{Role: RoleAssistant, Content: "blocked"}},
	}}
	hookReg := NewHookRegistry()
	_ = hookReg.Register("before_tool_execution", func(ctx HookContext) error {
		if tc := ctx.ToolCall(); tc != nil {
			ctx.Block("not allowed by policy")
		}
		return nil
	})
	agent := &Agent{
		Provider: provider, Tools: registry, MaxIterations: 5,
		Hooks: hookReg,
	}

	_, err := agent.Run(context.Background(), "write x")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(writeFile.args) != 0 {
		t.Errorf("tool executed %d times, want 0", len(writeFile.args))
	}
	second := provider.received[1]
	last := second[len(second)-1]
	if last.Role != RoleTool {
		t.Errorf("blocked result role = %q, want %q", last.Role, RoleTool)
	}
	if last.ToolCallID != "call-1" {
		t.Errorf("blocked result ToolCallID = %q, want %q", last.ToolCallID, "call-1")
	}
	if !json.Valid([]byte(last.Content)) {
		t.Errorf("blocked result content is not valid JSON: %q", last.Content)
	}
}

func TestLoopWithHooksAfterToolExecutionErrorDoesNotCorruptResult(t *testing.T) {
	// REQ-LOOP-15, "Hook error does not corrupt result": an after_tool_execution
	// handler that returns an error must not affect the tool result.
	readFile := &fakeTool{name: "read_file", result: "important data"}
	registry := NewRegistry()
	if err := registry.Register(readFile); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	provider := &fakeProvider{responses: [][]Message{
		{toolCall("call-1", "read_file", `{}`)},
		{{Role: RoleAssistant, Content: "done"}},
	}}
	hookReg := NewHookRegistry()
	_ = hookReg.Register("after_tool_execution", func(ctx HookContext) error {
		return errors.New("observer failure")
	})
	agent := &Agent{
		Provider: provider, Tools: registry, MaxIterations: 5,
		Hooks: hookReg,
	}

	answer, err := agent.Run(context.Background(), "read")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answer != "done" {
		t.Errorf("answer = %q, want %q", answer, "done")
	}
	// Verify the tool result was still delivered to the provider.
	second := provider.received[1]
	last := second[len(second)-1]
	if last.Role != RoleTool || last.Content != "important data" {
		t.Errorf("tool result = %+v, want role=tool content=important data", last)
	}
}
