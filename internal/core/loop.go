package core

import (
	"context"
	"encoding/json"
	"strings"
)

// Agent runs the conversation loop between a Provider and registered Tools
// (REQ-LOOP-1..4). MaxIterations bounds the number of provider calls before
// the loop terminates with an IterationLimitError (D7).
//
// The optional ports extend the loop without changing its termination
// contract (D14): Steering drains after each completed turn, FollowUp drains
// only when the loop would otherwise stop, Profiles applies queued profile
// switches between turns, and Permissions gates tool advertisement and
// dispatch. All four are nil-safe: leaving them unset reproduces the
// single-level loop exactly.
type Agent struct {
	Provider      Provider
	Tools         *Registry
	MaxIterations int

	// Steering drains after each completed turn; its messages are injected
	// before the next provider request (REQ-QUEUE-1). Nil disables it.
	Steering PendingQueue
	// FollowUp drains only when the loop would otherwise stop: after the
	// provider returns no tool calls and the steering queue is empty
	// (REQ-QUEUE-2). Nil disables it.
	FollowUp PendingQueue
	// Profiles applies queued profile switches between turns and returns the
	// system-prompt and marker messages to append (D16, REQ-LOOP-5/6).
	// Nil disables switching.
	Profiles ProfileManager
	// Permissions filters the advertised tool set before Chat and guards
	// dispatch with Allow (D15, REQ-PERM-3/4). Nil disables both.
	Permissions PermissionEvaluator
	// Observer is an optional port that receives turn and tool events
	// (REQ-LOOP-7). Nil disables event delivery; a panicking observer is
	// recovered by the emit helpers so the loop is never affected.
	Observer Observer
}

// Run executes one single-session loop from the user prompt to a final
// answer. It returns the provider's answer content, or a typed error when the
// loop must terminate: UnknownToolError for unregistered tools, PermissionError
// for denied tools, ToolError for tool execution failures, IterationLimitError
// when the budget is exhausted.
//
// The loop operates in two levels (D14): an inner level alternates provider
// requests with tool execution and steering-queue draining; an outer level
// drains the follow-up queue only when the inner level would otherwise stop.
// Queued messages are additive — they extend the turn sequence without
// changing the termination contract, and follow-up continuations still count
// against MaxIterations (REQ-QUEUE-1..3).
func (a *Agent) Run(ctx context.Context, prompt string) (string, error) {
	messages := []Message{{Role: RoleUser, Content: prompt}}

	for range a.MaxIterations {
		emitTurnStart(a.Observer)

		tools := a.Tools.List()
		if a.Permissions != nil {
			// D15: hide denied tools from the provider request payload so the
			// model never sees them (REQ-PERM-3).
			tools = a.Permissions.Filter(tools)
		}
		// REQ-LOOP-8: detect StreamingProvider via type assertion. If the
		// provider implements StreamingProvider, use the streaming path to
		// forward text deltas in real time. Otherwise fall back to Chat().
		var (
			response []Message
			err      error
		)
		if sp, ok := a.Provider.(StreamingProvider); ok {
			response, err = a.runStreamingTurn(ctx, sp, messages, tools)
		} else {
			response, err = a.Provider.Chat(ctx, messages, tools)
		}
		if err != nil {
			return "", err
		}
		messages = append(messages, response...)

		if !hasToolCalls(response) {
			// The inner level would stop: drain the steering queue first
			// (this turn completed without tools), then let the follow-up
			// queue keep the loop alive only when steering is empty.
			if a.Steering != nil {
				if drained := a.Steering.Drain(); len(drained) > 0 {
					var changed bool
					messages, changed, err = applySteering(ctx, a.Profiles, messages, drained)
					if err != nil {
						return "", err
					}
					if changed {
						emitTurnEnd(a.Observer)
						continue
					}
				}
			}
			if a.FollowUp != nil {
				if drained := a.FollowUp.Drain(); len(drained) > 0 {
					messages = append(messages, pendingMessages(drained)...)
					emitTurnEnd(a.Observer)
					continue
				}
			}
			emitTurnEnd(a.Observer)
			return lastContent(response), nil
		}

		for _, message := range response {
			call := message.ToolCall
			if call == nil {
				continue
			}
			tool, ok := a.Tools.Get(call.Name)
			if !ok {
				return "", &UnknownToolError{Name: call.Name}
			}
			if a.Permissions != nil && !a.Permissions.Allow(call.Name) {
				// D15, defense in depth: never execute a denied tool even if
				// a request arrives (REQ-PERM-4).
				return "", &PermissionError{Tool: call.Name}
			}
			emitToolCall(a.Observer, *call)
			result, err := tool.Execute(ctx, json.RawMessage(call.Arguments))
			if err != nil {
				return "", &ToolError{Name: call.Name, Err: err}
			}
			emitToolResult(a.Observer, call.ID, result)
			messages = append(messages, Message{
				Role:       RoleTool,
				Content:    result,
				ToolCallID: call.ID,
			})
		}

		// The turn completed with tool execution: drain the steering queue so
		// its messages are injected before the next provider request
		// (REQ-QUEUE-1). A failing tool returns above, so a failed turn never
		// drains and never injects queued messages (REQ-QUEUE-3). Queued
		// switch requests apply here, between turns — never mid-tool-call
		// (REQ-LOOP-5).
		if a.Steering != nil {
			if drained := a.Steering.Drain(); len(drained) > 0 {
				if messages, _, err = applySteering(ctx, a.Profiles, messages, drained); err != nil {
					return "", err
				}
			}
		}

		emitTurnEnd(a.Observer)
	}

	return "", &IterationLimitError{Max: a.MaxIterations}
}

// applySteering folds the drained steering messages into the conversation
// (REQ-QUEUE-1, REQ-LOOP-5/6). Content messages are injected as user messages
// in drain order; a queued SwitchProfile request is applied via the Profiles
// port and the returned messages (the new system prompt and the
// profile-context marker, D16) are appended after the user content. Multiple
// switch requests in one drain collapse to the last one so the final switch
// is deterministic (REQ-LOOP-5). changed reports whether the conversation
// grew; a drain with nothing injectable (no content, no switch, or a disabled
// Profiles port) changes nothing so the loop never spins.
func applySteering(ctx context.Context, profiles ProfileManager, messages []Message, drained []PendingMessage) ([]Message, bool, error) {
	var switchName string
	changed := false
	for _, p := range drained {
		if p.SwitchProfile != "" {
			switchName = p.SwitchProfile // the last switch wins (REQ-LOOP-5)
			continue
		}
		if p.Content != "" {
			messages = append(messages, Message{Role: RoleUser, Content: p.Content})
			changed = true
		}
	}
	if switchName != "" && profiles != nil {
		switched, err := profiles.ApplySwitch(ctx, switchName)
		if err != nil {
			return nil, false, err
		}
		messages = append(messages, switched...)
		changed = true
	}
	return messages, changed, nil
}

// pendingMessages maps drained queue messages onto the conversation as user
// messages (REQ-QUEUE-1). Steering drains fold switches separately through
// applySteering (D16); this remains the plain injection path for follow-up
// drains, which never carry switch requests.
func pendingMessages(pending []PendingMessage) []Message {
	messages := make([]Message, 0, len(pending))
	for _, p := range pending {
		messages = append(messages, Message{Role: RoleUser, Content: p.Content})
	}
	return messages
}

func hasToolCalls(messages []Message) bool {
	for _, message := range messages {
		if message.ToolCall != nil {
			return true
		}
	}
	return false
}

func lastContent(messages []Message) string {
	if len(messages) == 0 {
		return ""
	}
	return messages[len(messages)-1].Content
}

// runStreamingTurn calls StreamChat and consumes the channel, forwarding text
// deltas to the observer and accumulating tool calls. It returns the same
// message shape as Chat(): content messages plus tool-call messages. Tool
// calls are accumulated during streaming and executed only after the channel
// closes (REQ-LOOP-10, D8).
func (a *Agent) runStreamingTurn(ctx context.Context, sp StreamingProvider, messages []Message, tools []Tool) ([]Message, error) {
	stream, err := sp.StreamChat(ctx, messages, tools)
	if err != nil {
		return nil, err
	}

	var (
		textBuf      strings.Builder
		toolCalls    []*ToolCall
		pendingCalls = make(map[string]*ToolCall) // ID → tool call being built
	)

	for chunk := range stream {
		// D8: mid-stream error → return immediately.
		if chunk.Error != nil {
			return nil, chunk.Error
		}
		if chunk.Done {
			break
		}

		// REQ-LOOP-9: forward text deltas to the observer. emitTextDelta
		// is nil-safe and handles StreamingObserver type assertion.
		if chunk.TextDelta != "" {
			textBuf.WriteString(chunk.TextDelta)
			emitTextDelta(a.Observer, chunk.TextDelta)
		}

		// REQ-LOOP-10: accumulate tool calls from stream chunks.
		if chunk.ToolCallStart != nil {
			tc := &ToolCall{
				ID:   chunk.ToolCallStart.ID,
				Name: chunk.ToolCallStart.Name,
			}
			pendingCalls[tc.ID] = tc
			toolCalls = append(toolCalls, tc)
		}
		if chunk.ToolCallDelta != nil {
			if tc, ok := pendingCalls[chunk.ToolCallDelta.ID]; ok {
				tc.Arguments += chunk.ToolCallDelta.Arguments
			}
		}
	}

	// Build response messages matching Chat() shape: content message first,
	// then tool-call messages.
	var response []Message
	content := textBuf.String()
	if content != "" {
		response = append(response, Message{Role: RoleAssistant, Content: content})
	}
	for _, tc := range toolCalls {
		response = append(response, Message{
			Role:     RoleAssistant,
			ToolCall: tc,
		})
	}

	return response, nil
}
