package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/biggs-100/kui/internal/core"
)

// parseSSEStream reads an SSE response body and returns a channel of
// StreamChunks. It uses a 256KB scanner buffer per REQ-OAI-STREAM-6.
// The channel is buffered (64) per D3 and closed on completion or error.
func parseSSEStream(ctx context.Context, body io.Reader) <-chan core.StreamChunk {
	chunks := make(chan core.StreamChunk, 64)
	go func() {
		defer close(chunks)
		scanner := bufio.NewScanner(body)
		scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
		receivedDone := false
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				chunks <- core.StreamChunk{Error: ctx.Err()}
				return
			default:
			}
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				receivedDone = true
				chunks <- core.StreamChunk{Done: true}
				return
			}
			chunk, ok := parseSSEChunk(data)
			if ok {
				chunks <- chunk
			}
		}
		// After scanner exits: check context cancellation first
		if ctx.Err() != nil {
			chunks <- core.StreamChunk{Error: ctx.Err()}
			return
		}
		// Then scanner error (network failure, buffer overflow, etc.)
		if err := scanner.Err(); err != nil {
			chunks <- core.StreamChunk{Error: err}
			return
		}
		// Scanner reached EOF without [DONE] — premature connection close
		if !receivedDone {
			chunks <- core.StreamChunk{Error: io.ErrUnexpectedEOF}
		}
	}()
	return chunks
}

// streamChoice mirrors the OpenAI streaming response choice shape.
type streamChoice struct {
	Delta streamDelta `json:"delta"`
}

// streamDelta mirrors the OpenAI streaming delta shape.
type streamDelta struct {
	Content          string           `json:"content"`
	ReasoningContent string           `json:"reasoning_content"`
	ToolCalls        []streamToolCall `json:"tool_calls"`
}

// streamToolCall mirrors the OpenAI streaming tool call shape.
type streamToolCall struct {
	Index    int                `json:"index"`
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function streamToolFunction `json:"function"`
}

// streamToolFunction mirrors the OpenAI streaming tool call function shape.
type streamToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// streamResponse mirrors the full OpenAI streaming response shape.
type streamResponse struct {
	Choices []streamChoice `json:"choices"`
	Usage   *streamUsage   `json:"usage,omitempty"`
}

// streamUsage mirrors the OpenAI streaming usage shape.
type streamUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	PromptTokensDetails *promptTokensDetails `json:"prompt_tokens_details,omitempty"`
}

// promptTokensDetails contains cache information from OpenAI usage.
type promptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

// parseSSEChunk unmarshals a single SSE data payload into a StreamChunk.
// Returns (chunk, true) if the payload should be emitted, or (zero, false)
// if the payload is empty or contains no useful data.
func parseSSEChunk(data string) (core.StreamChunk, bool) {
	var resp streamResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return core.StreamChunk{}, false
	}

	// Usage chunk (no choices)
	if resp.Usage != nil && len(resp.Choices) == 0 {
		cached := 0
		if resp.Usage.PromptTokensDetails != nil {
			cached = resp.Usage.PromptTokensDetails.CachedTokens
		}
		return core.StreamChunk{
			Usage: &core.Usage{
				InputTokens:  resp.Usage.PromptTokens,
				OutputTokens: resp.Usage.CompletionTokens,
				TotalTokens:  resp.Usage.TotalTokens,
				CachedTokens: cached,
			},
		}, true
	}

	if len(resp.Choices) == 0 {
		return core.StreamChunk{}, false
	}

	delta := resp.Choices[0].Delta

	// Reasoning content delta (checked before text content)
	if delta.ReasoningContent != "" {
		return core.StreamChunk{ReasoningDelta: delta.ReasoningContent}, true
	}

	// Text content delta
	if delta.Content != "" {
		return core.StreamChunk{TextDelta: delta.Content}, true
	}

	// Tool call delta
	if len(delta.ToolCalls) > 0 {
		tc := delta.ToolCalls[0]
		// Start: function.name is set (first chunk of a tool call)
		if tc.Function.Name != "" {
			return core.StreamChunk{
				ToolCallStart: &core.ToolCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
				},
			}, true
		}
		// Delta: arguments are being accumulated
		if tc.Function.Arguments != "" {
			return core.StreamChunk{
				ToolCallDelta: &core.ToolCallDelta{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}, true
		}
	}

	return core.StreamChunk{}, false
}
