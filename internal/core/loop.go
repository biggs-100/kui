package core

import (
	"context"
	"encoding/json"
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
}

// Run executes one single-session loop from the user prompt to a final
// answer. It returns the provider's answer content, or a typed error when the
// loop must terminate: UnknownToolError for unregistered tools, ToolError for
// tool execution failures, IterationLimitError when the budget is exhausted.
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
		response, err := a.Provider.Chat(ctx, messages, a.Tools.List())
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
					messages = append(messages, pendingMessages(drained)...)
					continue
				}
			}
			if a.FollowUp != nil {
				if drained := a.FollowUp.Drain(); len(drained) > 0 {
					messages = append(messages, pendingMessages(drained)...)
					continue
				}
			}
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
			result, err := tool.Execute(ctx, json.RawMessage(call.Arguments))
			if err != nil {
				return "", &ToolError{Name: call.Name, Err: err}
			}
			messages = append(messages, Message{
				Role:       RoleTool,
				Content:    result,
				ToolCallID: call.ID,
			})
		}

		// The turn completed with tool execution: drain the steering queue so
		// its messages are injected before the next provider request
		// (REQ-QUEUE-1). A failing tool returns above, so a failed turn never
		// drains and never injects queued messages (REQ-QUEUE-3).
		if a.Steering != nil {
			messages = append(messages, pendingMessages(a.Steering.Drain())...)
		}
	}

	return "", &IterationLimitError{Max: a.MaxIterations}
}

// pendingMessages maps drained queue messages onto the conversation as user
// messages (REQ-QUEUE-1). Profile-switch messages are applied via the
// Profiles port when switching lands (D16).
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
