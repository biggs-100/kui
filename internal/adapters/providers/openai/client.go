// Package openai implements the OpenAI-compatible chat-completions provider
// adapter (REQ-PROV-1..4). Credentials come from the environment at
// construction time (D8); HTTP uses the standard library with a 60s timeout
// (D9) and a typed error surface (D10).
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/biggs-100/kui/internal/core"
)

const (
	// defaultBaseURL is used when OPENAI_BASE_URL is unset (REQ-PROV-3).
	defaultBaseURL = "https://api.openai.com/v1"
	// defaultModel is used when OPENAI_MODEL is unset so every request
	// carries an explicit model field.
	defaultModel = "gpt-4o-mini"
	// requestTimeout bounds every chat request (D9).
	requestTimeout = 60 * time.Second
)

// Client is a core.Provider speaking the OpenAI-compatible chat-completions
// protocol (REQ-PROV-1). It is safe for concurrent use.
type Client struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
}

// NewClient reads credentials and endpoint configuration from the
// environment: OPENAI_API_KEY is required and its absence is reported by
// naming the variable (D8, REQ-PROV-2); OPENAI_BASE_URL defaults to
// https://api.openai.com/v1 (REQ-PROV-3); OPENAI_MODEL defaults to
// gpt-4o-mini.
func NewClient() (*Client, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, errors.New("OPENAI_API_KEY is not set: export OPENAI_API_KEY before running kui")
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = defaultModel
	}
	return &Client{
		apiKey:  key,
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{Timeout: requestTimeout},
	}, nil
}

// SetModel reconfigures the model used by subsequent requests (D17,
// REQ-CLI-4). It is additive: construction behavior is unchanged, and every
// later request carries the new model field so the provider is reconfigured
// in place rather than rebuilt.
func (c *Client) SetModel(model string) {
	c.model = model
}

// Chat exchanges the message sequence and the advertised tool set with the
// provider and returns its response messages, which may carry tool calls
// (REQ-PROV-1). HTTP failures map to the typed error surface (D10); the API
// key never appears in any returned error (D8, REQ-PROV-3).
func (c *Client) Chat(ctx context.Context, messages []core.Message, tools []core.Tool) ([]core.Message, error) {
	body, err := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: requestMessages(messages),
		Tools:    requestTools(tools),
	})
	if err != nil {
		return nil, fmt.Errorf("build chat request: %w", err)
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &TransportError{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, &AuthError{}
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, &RateLimitError{}
	case resp.StatusCode >= http.StatusInternalServerError:
		return nil, &ServerError{Status: resp.StatusCode}
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("unexpected provider status %d", resp.StatusCode)
	}

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, &ParseError{Err: err}
	}
	return parseResponse(parsed)
}

// StreamChat sends a streaming chat request and returns a channel of
// StreamChunks. It implements the StreamingProvider interface (REQ-OAI-STREAM-1).
// The request carries "stream": true and the response is parsed as SSE.
func (c *Client) StreamChat(ctx context.Context, messages []core.Message, tools []core.Tool) (<-chan core.StreamChunk, error) {
	body, err := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: requestMessages(messages),
		Tools:    requestTools(tools),
		Stream:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("build stream request: %w", err)
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &TransportError{Err: err}
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		resp.Body.Close()
		return nil, &AuthError{}
	case resp.StatusCode == http.StatusTooManyRequests:
		resp.Body.Close()
		return nil, &RateLimitError{}
	case resp.StatusCode >= http.StatusInternalServerError:
		resp.Body.Close()
		return nil, &ServerError{Status: resp.StatusCode}
	case resp.StatusCode != http.StatusOK:
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected provider status %d", resp.StatusCode)
	}

	// A non-SSE response means the server ignored "stream": true and
	// returned a regular JSON response. Parse it synchronously and emit the
	// messages as a single chunk sequence (text + tool calls + done) so the
	// caller never needs a second HTTP request (REQ-OAI-STREAM-1). Mock
	// servers returning application/json work through the streaming path.
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		defer func() { _ = resp.Body.Close() }()
		var parsed chatResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return nil, &ParseError{Err: err}
		}
		msgs, err := parseResponse(parsed)
		if err != nil {
			return nil, err
		}
		return chunksFromMessages(msgs), nil
	}

	return parseSSEStream(ctx, resp.Body), nil
}

// chunksFromMessages converts a synchronous response message set into a
// StreamChunk sequence: each content message becomes a text delta, each tool
// call becomes a ToolCallStart + ToolCallDelta pair, and the sequence ends
// with a Done chunk. It is used when a streaming request receives a regular
// JSON response (the server ignored "stream": true).
func chunksFromMessages(msgs []core.Message) <-chan core.StreamChunk {
	chunks := make(chan core.StreamChunk, 64)
	go func() {
		defer close(chunks)
		for _, m := range msgs {
			if m.Content != "" {
				chunks <- core.StreamChunk{TextDelta: m.Content}
			}
			if m.ToolCall != nil {
				chunks <- core.StreamChunk{
					ToolCallStart: &core.ToolCall{
						ID:   m.ToolCall.ID,
						Name: m.ToolCall.Name,
					},
				}
				chunks <- core.StreamChunk{
					ToolCallDelta: &core.ToolCallDelta{
						ID:        m.ToolCall.ID,
						Name:      m.ToolCall.Name,
						Arguments: m.ToolCall.Arguments,
					},
				}
			}
		}
		chunks <- core.StreamChunk{Done: true}
	}()
	return chunks
}

// chatRequest is the OpenAI-compatible chat completions request body. Model is
// always present so compatible servers never receive a request without it.
type chatRequest struct {
	Model    string           `json:"model"`
	Messages []requestMessage `json:"messages"`
	Tools    []requestTool    `json:"tools,omitempty"`
	Stream   bool             `json:"stream,omitempty"`
}

type requestMessage struct {
	Role       string            `json:"role"`
	Content    string            `json:"content"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	ToolCalls  []requestToolCall `json:"tool_calls,omitempty"`
}

type requestToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Function requestToolCallFunc `json:"function"`
}

type requestToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type requestTool struct {
	Type     string          `json:"type"`
	Function requestToolFunc `json:"function"`
}

type requestToolFunc struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// chatResponse is the OpenAI-compatible chat completions response body.
type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message responseMessage `json:"message"`
}

type responseMessage struct {
	Role      string             `json:"role"`
	Content   string             `json:"content"`
	ToolCalls []responseToolCall `json:"tool_calls"`
}

type responseToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function responseFunc `json:"function"`
}

type responseFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// requestMessages maps the core conversation to the wire format. Assistant
// tool calls become a tool_calls entry; tool results carry their call ID.
func requestMessages(messages []core.Message) []requestMessage {
	out := make([]requestMessage, 0, len(messages))
	for _, m := range messages {
		rm := requestMessage{Role: m.Role, Content: m.Content}
		if m.ToolCallID != "" {
			rm.ToolCallID = m.ToolCallID
		}
		if m.ToolCall != nil {
			rm.ToolCalls = []requestToolCall{{
				ID:   m.ToolCall.ID,
				Type: "function",
				Function: requestToolCallFunc{
					Name:      m.ToolCall.Name,
					Arguments: m.ToolCall.Arguments,
				},
			}}
		}
		out = append(out, rm)
	}
	return out
}

// requestTools maps the advertised tool set to function tools, embedding each
// tool's raw JSON schema as the parameters (D3).
func requestTools(tools []core.Tool) []requestTool {
	out := make([]requestTool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, requestTool{
			Type: "function",
			Function: requestToolFunc{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  json.RawMessage(tool.Schema()),
			},
		})
	}
	return out
}

// parseResponse maps the provider response to core messages. An assistant
// message with several tool calls becomes one message per call so the loop can
// dispatch each independently.
func parseResponse(resp chatResponse) ([]core.Message, error) {
	if len(resp.Choices) == 0 {
		return nil, &ParseError{Err: errors.New("response contains no choices")}
	}
	messages := make([]core.Message, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		msg := core.Message{Role: choice.Message.Role, Content: choice.Message.Content}
		if len(choice.Message.ToolCalls) == 0 {
			messages = append(messages, msg)
			continue
		}
		for _, tc := range choice.Message.ToolCalls {
			msg.ToolCall = &core.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

// AuthError reports HTTP 401 from the provider (D10, REQ-PROV-4).
type AuthError struct{}

func (e *AuthError) Error() string {
	return "provider authentication failed (401)"
}

// RateLimitError reports HTTP 429 from the provider (D10, REQ-PROV-4).
type RateLimitError struct{}

func (e *RateLimitError) Error() string {
	return "provider rate limit exceeded (429)"
}

// ServerError reports HTTP 5xx from the provider (D10, REQ-PROV-4).
type ServerError struct {
	Status int
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("provider server error (%d)", e.Status)
}

// TransportError reports a network-level failure such as a refused connection
// (D10, REQ-PROV-4).
type TransportError struct {
	Err error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("provider request failed: %v", e.Err)
}

// Unwrap exposes the underlying transport error.
func (e *TransportError) Unwrap() error {
	return e.Err
}

// ParseError reports a response body that could not be interpreted (D10,
// REQ-PROV-4).
type ParseError struct {
	Err error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("invalid provider response: %v", e.Err)
}

// Unwrap exposes the underlying parse error.
func (e *ParseError) Unwrap() error {
	return e.Err
}
