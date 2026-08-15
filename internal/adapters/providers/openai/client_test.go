package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

const testAPIKey = "sk-test-123"

// fakeTool is a minimal core.Tool used to exercise the wire format of the
// tools advertisement in the request body.
type fakeTool struct {
	name, description, schema string
}

func (f fakeTool) Name() string        { return f.name }
func (f fakeTool) Description() string { return f.description }
func (f fakeTool) Schema() string      { return f.schema }
func (f fakeTool) Execute(context.Context, json.RawMessage) (string, error) {
	return "", nil
}

func readFileTool() fakeTool {
	return fakeTool{
		name:        "read_file",
		description: "Read the full text content of a file inside the workspace",
		schema:      `{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`,
	}
}

// newClientEnv sets the credentials env vars and constructs a client pointing
// at srv. The base URL is always set to srv.URL so tests never touch the
// network; callers override OPENAI_BASE_URL before invoking when needed.
func newClientEnv(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	t.Setenv("OPENAI_API_KEY", testAPIKey)
	t.Setenv("OPENAI_BASE_URL", srv.URL)
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return c
}

func okServer(t *testing.T, assert func(t *testing.T, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if assert != nil {
			assert(t, r)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"hello from server"}}]}`)
	}))
}

// decodedRequest mirrors the OpenAI-compatible chat request wire format so the
// tests assert the actual bytes the client sends, not internal structs.
type decodedRequest struct {
	Messages []struct {
		Role       string `json:"role"`
		Content    string `json:"content"`
		ToolCallID string `json:"tool_call_id"`
		ToolCalls  []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	} `json:"messages"`
	Tools []struct {
		Type     string `json:"type"`
		Function struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		} `json:"function"`
	} `json:"tools"`
}

func decodeRequest(t *testing.T, r *http.Request) decodedRequest {
	t.Helper()
	var got decodedRequest
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return got
}

// TestChatToolCall covers REQ-PROV-1 "Response with tool call" and
// REQ-PROV-2 "Key present": the client POSTs messages and tool definitions to
// {base}/chat/completions with the key as Bearer token, and returns the tool
// call to the loop.
func TestChatToolCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testAPIKey {
			t.Errorf("Authorization = %q, want Bearer %s", got, testAPIKey)
		}

		req := decodeRequest(t, r)
		if len(req.Messages) != 3 {
			t.Fatalf("messages = %d, want 3", len(req.Messages))
		}
		if req.Messages[0].Role != core.RoleUser || req.Messages[0].Content != "list the files" {
			t.Errorf("messages[0] = %+v, want user message %q", req.Messages[0], "list the files")
		}
		if req.Messages[1].Role != core.RoleAssistant {
			t.Errorf("messages[1].role = %q, want assistant", req.Messages[1].Role)
		}
		if len(req.Messages[1].ToolCalls) != 1 {
			t.Fatalf("messages[1].tool_calls = %d, want 1", len(req.Messages[1].ToolCalls))
		}
		tc := req.Messages[1].ToolCalls[0]
		if tc.ID != "call_1" || tc.Type != "function" {
			t.Errorf("tool call = %+v, want id call_1 type function", tc)
		}
		if tc.Function.Name != "read_file" || tc.Function.Arguments != `{"path":"a.txt"}` {
			t.Errorf("tool call function = %+v, want read_file with path argument", tc.Function)
		}
		if req.Messages[2].Role != core.RoleTool || req.Messages[2].ToolCallID != "call_1" || req.Messages[2].Content != "contents of a.txt" {
			t.Errorf("messages[2] = %+v, want tool result for call_1", req.Messages[2])
		}
		if len(req.Tools) != 1 {
			t.Fatalf("tools = %d, want 1", len(req.Tools))
		}
		if req.Tools[0].Type != "function" || req.Tools[0].Function.Name != "read_file" || req.Tools[0].Function.Description == "" {
			t.Errorf("tools[0] = %+v, want function tool read_file", req.Tools[0])
		}
		var params map[string]any
		if err := json.Unmarshal(req.Tools[0].Function.Parameters, &params); err != nil {
			t.Fatalf("parameters is not valid JSON: %v", err)
		}
		if params["type"] != "object" {
			t.Errorf("parameters.type = %v, want object", params["type"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"a.txt\"}"}}]}}]}`)
	}))
	defer srv.Close()
	c := newClientEnv(t, srv)

	messages, err := c.Chat(context.Background(), []core.Message{
		{Role: core.RoleUser, Content: "list the files"},
		{Role: core.RoleAssistant, ToolCall: &core.ToolCall{ID: "call_1", Name: "read_file", Arguments: `{"path":"a.txt"}`}},
		{Role: core.RoleTool, Content: "contents of a.txt", ToolCallID: "call_1"},
	}, []core.Tool{readFileTool()})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}
	m := messages[0]
	if m.Role != core.RoleAssistant {
		t.Errorf("role = %q, want assistant", m.Role)
	}
	if m.ToolCall == nil {
		t.Fatal("ToolCall = nil, want the tool call from the response")
	}
	if m.ToolCall.ID != "call_1" || m.ToolCall.Name != "read_file" || m.ToolCall.Arguments != `{"path":"a.txt"}` {
		t.Errorf("ToolCall = %+v, want call_1 read_file with path argument", m.ToolCall)
	}
}

// TestChatPlainAnswer is the triangulation of REQ-PROV-1: a response without
// tool calls yields a plain content message.
func TestChatPlainAnswer(t *testing.T) {
	srv := okServer(t, nil)
	defer srv.Close()
	c := newClientEnv(t, srv)

	messages, err := c.Chat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}
	if messages[0].Content != "hello from server" {
		t.Errorf("content = %q, want %q", messages[0].Content, "hello from server")
	}
	if messages[0].ToolCall != nil {
		t.Errorf("ToolCall = %+v, want nil", messages[0].ToolCall)
	}
}

// TestChatMultipleToolCalls triangulates the tool-call mapping: several
// tool_calls in one response message become one message per call.
func TestChatMultipleToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[
			{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"a.txt\"}"}},
			{"id":"call_2","type":"function","function":{"name":"bash","arguments":"{\"command\":\"ls\"}"}}
		]}}]}`)
	}))
	defer srv.Close()
	c := newClientEnv(t, srv)

	messages, err := c.Chat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "do both"}}, nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(messages))
	}
	if messages[0].ToolCall == nil || messages[0].ToolCall.ID != "call_1" || messages[0].ToolCall.Name != "read_file" {
		t.Errorf("messages[0].ToolCall = %+v, want call_1 read_file", messages[0].ToolCall)
	}
	if messages[1].ToolCall == nil || messages[1].ToolCall.ID != "call_2" || messages[1].ToolCall.Name != "bash" {
		t.Errorf("messages[1].ToolCall = %+v, want call_2 bash", messages[1].ToolCall)
	}
}

// TestChatMalformedBody covers REQ-PROV-1 "Malformed response body": invalid
// JSON is surfaced as a typed parse error.
func TestChatMalformedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `not json at all`)
	}))
	defer srv.Close()
	c := newClientEnv(t, srv)

	_, err := c.Chat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %v, want *ParseError", err)
	}
}

// TestChatEmptyChoices triangulates the parse path: a well-formed body without
// choices is also a parse error.
func TestChatEmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[]}`)
	}))
	defer srv.Close()
	c := newClientEnv(t, srv)

	_, err := c.Chat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %v, want *ParseError", err)
	}
}

// TestChatAuthError covers REQ-PROV-4 "Authentication failure": HTTP 401 maps
// to AuthError, and the key must not leak into the error even when the server
// echoes it back (REQ-PROV-3).
func TestChatAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":{"message":"invalid key `+testAPIKey+`"}}`)
	}))
	defer srv.Close()
	c := newClientEnv(t, srv)

	_, err := c.Chat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("error = %v, want *AuthError", err)
	}
	if strings.Contains(err.Error(), testAPIKey) {
		t.Errorf("error %q must not contain the API key", err)
	}
}

// TestChatRateLimitError covers REQ-PROV-4: HTTP 429 maps to RateLimitError.
func TestChatRateLimitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":{"message":"slow down"}}`)
	}))
	defer srv.Close()
	c := newClientEnv(t, srv)

	_, err := c.Chat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
	var rateErr *RateLimitError
	if !errors.As(err, &rateErr) {
		t.Fatalf("error = %v, want *RateLimitError", err)
	}
}

// TestChatServerError covers REQ-PROV-4 "Server error": HTTP 5xx maps to
// ServerError carrying the status code.
func TestChatServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":{"message":"boom"}}`)
	}))
	defer srv.Close()
	c := newClientEnv(t, srv)

	_, err := c.Chat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
	var srvErr *ServerError
	if !errors.As(err, &srvErr) {
		t.Fatalf("error = %v, want *ServerError", err)
	}
	if srvErr.Status != http.StatusInternalServerError {
		t.Errorf("ServerError.Status = %d, want 500", srvErr.Status)
	}
}

// TestChatTransportError covers REQ-PROV-4 "Transport failure": an unreachable
// endpoint maps to TransportError.
func TestChatTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close() // nothing listens on addr anymore

	t.Setenv("OPENAI_API_KEY", testAPIKey)
	t.Setenv("OPENAI_BASE_URL", addr)
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = c.Chat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
	var transportErr *TransportError
	if !errors.As(err, &transportErr) {
		t.Fatalf("error = %v, want *TransportError", err)
	}
}

// TestNewClientKeyMissing covers REQ-PROV-2 "Key missing": creation fails with
// an actionable error naming OPENAI_API_KEY (D8).
func TestNewClientKeyMissing(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "https://example.com/v1")

	_, err := NewClient()
	if err == nil {
		t.Fatal("NewClient() error = nil, want failure naming OPENAI_API_KEY")
	}
	if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("error %q must name OPENAI_API_KEY", err)
	}
}

// TestChatKeyPresent covers REQ-PROV-2 "Key present": the request carries the
// key as a Bearer token.
func TestChatKeyPresent(t *testing.T) {
	srv := okServer(t, func(t *testing.T, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+testAPIKey {
			t.Errorf("Authorization = %q, want Bearer %s", got, testAPIKey)
		}
	})
	defer srv.Close()
	c := newClientEnv(t, srv)

	if _, err := c.Chat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
}

// TestChatCustomBaseURL covers REQ-PROV-3 "Custom base URL": the request
// targets OPENAI_BASE_URL (REQ-PROV-1).
func TestChatCustomBaseURL(t *testing.T) {
	var hit bool
	srv := okServer(t, func(t *testing.T, r *http.Request) {
		hit = true
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
	})
	defer srv.Close()
	c := newClientEnv(t, srv)

	if _, err := c.Chat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if !hit {
		t.Error("request did not reach the custom base URL server")
	}
}

// captureTransport records the request URL and returns a canned chat response,
// letting tests observe the target URL without any network access.
type captureTransport struct {
	gotURL *url.URL
}

func (tr *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr.gotURL = req.URL
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`)),
		Request:    req,
	}, nil
}

// TestChatDefaultBaseURL covers REQ-PROV-3 "Default base URL": with
// OPENAI_BASE_URL unset the request targets
// https://api.openai.com/v1/chat/completions.
func TestChatDefaultBaseURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", testAPIKey)
	t.Setenv("OPENAI_BASE_URL", "")
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	transport := &captureTransport{}
	c.http = &http.Client{Transport: transport}

	if _, err := c.Chat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if transport.gotURL == nil {
		t.Fatal("no request captured")
	}
	if want := "https://api.openai.com/v1/chat/completions"; transport.gotURL.String() != want {
		t.Errorf("request URL = %q, want %q", transport.gotURL.String(), want)
	}
}
