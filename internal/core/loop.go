package core

import (
	"context"
	"encoding/json"
)

// Agent runs the conversation loop between a Provider and registered Tools
// (REQ-LOOP-1..4). MaxIterations bounds the number of provider calls before
// the loop terminates with an IterationLimitError (D7).
type Agent struct {
	Provider      Provider
	Tools         *Registry
	MaxIterations int
}

// Run executes one single-session loop from the user prompt to a final
// answer. It returns the provider's answer content, or a typed error when the
// loop must terminate: UnknownToolError for unregistered tools, ToolError for
// tool execution failures, IterationLimitError when the budget is exhausted.
func (a *Agent) Run(ctx context.Context, prompt string) (string, error) {
	messages := []Message{{Role: RoleUser, Content: prompt}}

	for range a.MaxIterations {
		response, err := a.Provider.Chat(ctx, messages, a.Tools.List())
		if err != nil {
			return "", err
		}
		messages = append(messages, response...)

		if !hasToolCalls(response) {
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
	}

	return "", &IterationLimitError{Max: a.MaxIterations}
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
