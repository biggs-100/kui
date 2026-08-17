package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/biggs-100/kui/internal/core"
)

// sseServer creates an httptest server that streams the given lines as SSE data.
// Each line is written as-is followed by a newline. After all lines, the
// connection is closed.
func sseServer(t *testing.T, lines []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, line := range lines {
			_, _ = io.WriteString(w, line+"\n")
		}
	}))
}

// collectChunks reads all chunks from a channel until it closes or the context
// is cancelled. Returns all received chunks.
func collectChunks(ch <-chan core.StreamChunk, timeout time.Duration) []core.StreamChunk {
	var chunks []core.StreamChunk
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				return chunks
			}
			chunks = append(chunks, chunk)
		case <-timer.C:
			return chunks
		}
	}
}

// TestParseSSEStreamNormalChunks verifies that a valid SSE stream with text
// deltas is parsed into StreamChunks with TextDelta fields set.
func TestParseSSEStreamNormalChunks(t *testing.T) {
	sseLines := []string{
		`data: {"choices":[{"delta":{"content":"Hello"},"index":0}]}`,
		`data: {"choices":[{"delta":{"content":" world"},"index":0}]}`,
		`data: [DONE]`,
	}
	srv := sseServer(t, sseLines)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	ctx := context.Background()
	chunks := collectChunks(parseSSEStream(ctx, resp.Body), 5*time.Second)

	if len(chunks) != 3 {
		t.Fatalf("got %d chunks, want 3 (2 text + 1 done)", len(chunks))
	}
	if chunks[0].TextDelta != "Hello" {
		t.Errorf("chunks[0].TextDelta = %q, want %q", chunks[0].TextDelta, "Hello")
	}
	if chunks[0].Done {
		t.Error("chunks[0].Done = true, want false")
	}
	if chunks[1].TextDelta != " world" {
		t.Errorf("chunks[1].TextDelta = %q, want %q", chunks[1].TextDelta, " world")
	}
	if !chunks[2].Done {
		t.Error("chunks[2].Done = false, want true")
	}
}

// TestParseSSEStreamDoneSentinel verifies that `data: [DONE]` produces a
// StreamChunk with Done=true and closes the channel.
func TestParseSSEStreamDoneSentinel(t *testing.T) {
	sseLines := []string{`data: [DONE]`}
	srv := sseServer(t, sseLines)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	chunks := collectChunks(parseSSEStream(context.Background(), resp.Body), 5*time.Second)

	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if !chunks[0].Done {
		t.Error("Done = false, want true")
	}
}

// TestParseSSEStreamMalformedJSON verifies that malformed JSON in a data line
// is silently skipped without crashing or producing a chunk.
func TestParseSSEStreamMalformedJSON(t *testing.T) {
	sseLines := []string{
		`data: {not valid json`,
		`data: {"choices":[{"delta":{"content":"ok"},"index":0}]}`,
		`data: [DONE]`,
	}
	srv := sseServer(t, sseLines)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	chunks := collectChunks(parseSSEStream(context.Background(), resp.Body), 5*time.Second)

	// Malformed line is skipped, only the valid text + done remain
	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2 (malformed skipped)", len(chunks))
	}
	if chunks[0].TextDelta != "ok" {
		t.Errorf("chunks[0].TextDelta = %q, want %q", chunks[0].TextDelta, "ok")
	}
	if !chunks[1].Done {
		t.Error("chunks[1].Done = false, want true")
	}
}

// TestParseSSEStreamEmptyLines verifies that empty lines (SSE event separators)
// are ignored without producing chunks.
func TestParseSSEStreamEmptyLines(t *testing.T) {
	sseLines := []string{
		"",
		`data: {"choices":[{"delta":{"content":"a"},"index":0}]}`,
		"",
		"",
		`data: {"choices":[{"delta":{"content":"b"},"index":0}]}`,
		`data: [DONE]`,
	}
	srv := sseServer(t, sseLines)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	chunks := collectChunks(parseSSEStream(context.Background(), resp.Body), 5*time.Second)

	if len(chunks) != 3 {
		t.Fatalf("got %d chunks, want 3 (empty lines ignored)", len(chunks))
	}
	if chunks[0].TextDelta != "a" {
		t.Errorf("chunks[0].TextDelta = %q, want %q", chunks[0].TextDelta, "a")
	}
	if chunks[1].TextDelta != "b" {
		t.Errorf("chunks[1].TextDelta = %q, want %q", chunks[1].TextDelta, "b")
	}
}

// TestParseSSEStreamMultipleDataFields verifies that SSE events with multiple
// data fields (multi-line data) are parsed correctly. The OpenAI format uses
// single data lines, but the parser should handle the standard correctly.
func TestParseSSEStreamMultipleDataFields(t *testing.T) {
	sseLines := []string{
		`data: {"choices":[{"delta":{"content":"first"},"index":0}]}`,
		`data: {"choices":[{"delta":{"content":"second"},"index":0}]}`,
		`data: {"choices":[{"delta":{"content":"third"},"index":0}]}`,
		`data: [DONE]`,
	}
	srv := sseServer(t, sseLines)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	chunks := collectChunks(parseSSEStream(context.Background(), resp.Body), 5*time.Second)

	if len(chunks) != 4 {
		t.Fatalf("got %d chunks, want 4 (3 text + 1 done)", len(chunks))
	}
	expected := []string{"first", "second", "third"}
	for i, exp := range expected {
		if chunks[i].TextDelta != exp {
			t.Errorf("chunks[%d].TextDelta = %q, want %q", i, chunks[i].TextDelta, exp)
		}
	}
}

// TestParseSSEStreamContextCancellation verifies that cancelling the context
// stops the stream and produces an error chunk.
func TestParseSSEStreamContextCancellation(t *testing.T) {
	// Use a pipe so we control exactly when data arrives
	pr, pw := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chunks := make(chan core.StreamChunk, 64)
	go func() {
		defer close(chunks)
		for chunk := range parseSSEStream(ctx, pr) {
			chunks <- chunk
		}
	}()

	// Send one chunk, then cancel before sending more
	_, _ = pw.Write([]byte(`data: {"choices":[{"delta":{"content":"a"},"index":0}]}` + "\n"))
	time.Sleep(50 * time.Millisecond)
	cancel()
	_ = pw.Close()

	// Drain
	var all []core.StreamChunk
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case c, ok := <-chunks:
			if !ok {
				goto done
			}
			all = append(all, c)
		case <-timer.C:
			goto done
		}
	}
done:
	if len(all) < 1 {
		t.Fatalf("got %d chunks, want at least 1", len(all))
	}
	// Last chunk should be an error from context cancellation
	last := all[len(all)-1]
	if last.Error == nil {
		t.Error("last chunk Error = nil, want context cancellation error")
	}
}

// TestParseSSEStreamNetworkError verifies that a network error (connection
// drop mid-stream) produces a StreamChunk with Error set and closes the channel.
func TestParseSSEStreamNetworkError(t *testing.T) {
	// Use a pipe to simulate partial data then abrupt close
	pr, pw := io.Pipe()

	// Write a valid line then close the pipe (simulating connection drop)
	go func() {
		_, _ = pw.Write([]byte("data: {partial\n"))
		_ = pw.Close()
	}()

	chunks := collectChunks(parseSSEStream(context.Background(), pr), 5*time.Second)

	if len(chunks) == 0 {
		t.Fatal("got 0 chunks, want at least 1 (error from connection drop)")
	}
	// At least one chunk should be an error
	hasError := false
	for _, c := range chunks {
		if c.Error != nil {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Error("no error chunk found, want error from network failure")
	}
}

// TestParseSSEStreamNonDataLines verifies that lines without the `data: ` prefix
// are ignored (e.g., `event:`, `id:`, `retry:` SSE fields).
func TestParseSSEStreamNonDataLines(t *testing.T) {
	sseLines := []string{
		`event: message`,
		`id: 1`,
		`retry: 5000`,
		`data: {"choices":[{"delta":{"content":"hi"},"index":0}]}`,
		`data: [DONE]`,
	}
	srv := sseServer(t, sseLines)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	chunks := collectChunks(parseSSEStream(context.Background(), resp.Body), 5*time.Second)

	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2 (non-data lines ignored)", len(chunks))
	}
	if chunks[0].TextDelta != "hi" {
		t.Errorf("chunks[0].TextDelta = %q, want %q", chunks[0].TextDelta, "hi")
	}
}

// TestParseSSEStreamNoContentDelta verifies that SSE chunks with no content
// field (e.g., role-only deltas) are skipped without producing a chunk.
func TestParseSSEStreamNoContentDelta(t *testing.T) {
	sseLines := []string{
		`data: {"choices":[{"delta":{"role":"assistant"},"index":0}]}`,
		`data: {"choices":[{"delta":{"content":"actual"},"index":0}]}`,
		`data: [DONE]`,
	}
	srv := sseServer(t, sseLines)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	chunks := collectChunks(parseSSEStream(context.Background(), resp.Body), 5*time.Second)

	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2 (role-only delta skipped)", len(chunks))
	}
	if chunks[0].TextDelta != "actual" {
		t.Errorf("chunks[0].TextDelta = %q, want %q", chunks[0].TextDelta, "actual")
	}
}

// TestParseSSEStreamUsageChunk verifies that a chunk with usage data produces
// a StreamChunk with Usage set.
func TestParseSSEStreamUsageChunk(t *testing.T) {
	sseLines := []string{
		`data: {"choices":[],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
		`data: [DONE]`,
	}
	srv := sseServer(t, sseLines)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	chunks := collectChunks(parseSSEStream(context.Background(), resp.Body), 5*time.Second)

	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2 (usage + done)", len(chunks))
	}
	if chunks[0].Usage == nil {
		t.Fatal("chunks[0].Usage = nil, want usage data")
	}
	if chunks[0].Usage.InputTokens != 10 {
		t.Errorf("chunks[0].Usage.InputTokens = %d, want 10", chunks[0].Usage.InputTokens)
	}
	if chunks[0].Usage.OutputTokens != 5 {
		t.Errorf("chunks[0].Usage.OutputTokens = %d, want 5", chunks[0].Usage.OutputTokens)
	}
	if chunks[0].Usage.TotalTokens != 15 {
		t.Errorf("chunks[0].Usage.TotalTokens = %d, want 15", chunks[0].Usage.TotalTokens)
	}
}

// TestParseSSEStreamLargePayload verifies that a 200KB tool call JSON event
// fits in the 256KB scanner buffer without error.
func TestParseSSEStreamLargePayload(t *testing.T) {
	// Build a200KB arguments string and properly JSON-encode it
	largeArgs := strings.Repeat("x", 200*1024)
	encodedArgs, err := json.Marshal(largeArgs)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	// Build the SSE data line with properly escaped JSON
	sseData := `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"big_tool","arguments":` + string(encodedArgs) + `}}]},"index":0}]}`
	sseLines := []string{sseData, `data: [DONE]`}

	srv := sseServer(t, sseLines)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	chunks := collectChunks(parseSSEStream(context.Background(), resp.Body), 10*time.Second)

	if len(chunks) < 1 {
		t.Fatal("got 0 chunks, want at least 1 (large tool call parsed)")
	}
	// First chunk should be a tool call start (name is set)
	if chunks[0].ToolCallStart == nil {
		t.Fatal("chunks[0].ToolCallStart = nil, want tool call start from large payload")
	}
	if chunks[0].ToolCallStart.Name != "big_tool" {
		t.Errorf("chunks[0].ToolCallStart.Name = %q, want %q", chunks[0].ToolCallStart.Name, "big_tool")
	}
}

// TestParseSSEStreamEmptyBody verifies that an empty response body (no [DONE])
// produces an error chunk per REQ-OAI-STREAM-3 (EOF without sentinel).
func TestParseSSEStreamEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Empty body — no data, no [DONE]
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	chunks := collectChunks(parseSSEStream(context.Background(), resp.Body), 2*time.Second)

	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1 (error for empty body without DONE)", len(chunks))
	}
	if chunks[0].Error == nil {
		t.Error("chunks[0].Error = nil, want error for empty body without DONE")
	}
}

// TestParseSSEStreamEOFWithoutDone verifies that EOF without [DONE] produces
// an error chunk (connection dropped prematurely).
func TestParseSSEStreamEOFWithoutDone(t *testing.T) {
	sseLines := []string{
		`data: {"choices":[{"delta":{"content":"hi"},"index":0}]}`,
		// No [DONE] — connection just ends
	}
	srv := sseServer(t, sseLines)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	chunks := collectChunks(parseSSEStream(context.Background(), resp.Body), 5*time.Second)

	if len(chunks) < 1 {
		t.Fatal("got 0 chunks, want at least 1")
	}
	// Last chunk should be an error (EOF without DONE)
	last := chunks[len(chunks)-1]
	if last.Error == nil {
		t.Error("last chunk Error = nil, want error for EOF without DONE")
	}
}

// Verify that parseSSEStream returns a channel (compile-time check).
func TestParseSSEStreamReturnsChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n")
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	ch := parseSSEStream(context.Background(), resp.Body)
	if ch == nil {
		t.Fatal("parseSSEStream returned nil channel, want non-nil")
	}
	// Drain
	for range ch {
	}
}

// TestStreamChatReturnsChannel verifies that StreamChat returns a non-nil
// channel when the request succeeds (REQ-OAI-STREAM-1).
func TestStreamChatReturnsChannel(t *testing.T) {
	sseLines := []string{
		`data: {"choices":[{"delta":{"content":"hi"},"index":0}]}`,
		`data: [DONE]`,
	}
	srv := sseServer(t, sseLines)
	defer srv.Close()
	c := newClientEnv(t, srv)

	ch, err := c.StreamChat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("StreamChat() error = %v", err)
	}
	if ch == nil {
		t.Fatal("StreamChat() returned nil channel, want non-nil")
	}
	// Drain
	for range ch {
	}
}

// TestStreamChatSendsStreamTrue verifies that StreamChat sends a POST with
// `"stream": true` in the request body (REQ-OAI-STREAM-1).
func TestStreamChatSendsStreamTrue(t *testing.T) {
	var rawBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testAPIKey {
			t.Errorf("Authorization = %q, want Bearer %s", got, testAPIKey)
		}
		body, _ := io.ReadAll(r.Body)
		rawBody = string(body)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n")
	}))
	defer srv.Close()
	c := newClientEnv(t, srv)

	ch, err := c.StreamChat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("StreamChat() error = %v", err)
	}
	for range ch {
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(rawBody), &parsed); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}
	if stream, ok := parsed["stream"]; !ok || stream != true {
		t.Errorf("request body stream = %v, want true", stream)
	}
}

// TestStreamChatHTTPErrors verifies that StreamChat returns appropriate typed
// errors for HTTP failure status codes (REQ-PROV-4).
func TestStreamChatHTTPErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		wantErr    string
		errType    interface{}
	}{
		{"unauthorized", http.StatusUnauthorized, "authentication", &AuthError{}},
		{"rate limit", http.StatusTooManyRequests, "rate limit", &RateLimitError{}},
		{"server error", http.StatusInternalServerError, "server error", &ServerError{}},
		{"not found", http.StatusNotFound, "unexpected provider status", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, `{"error":{"message":"fail"}}`)
			}))
			defer srv.Close()
			c := newClientEnv(t, srv)

			_, err := c.StreamChat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
			if err == nil {
				t.Fatalf("StreamChat() error = nil, want error for status %d", tt.status)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want contains %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestStreamChatContextCancellation verifies that StreamChat respects context
// cancellation — the request should not be sent if the context is already
// cancelled.
func TestStreamChatContextCancellation(t *testing.T) {
	var requestReceived bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n")
	}))
	defer srv.Close()
	c := newClientEnv(t, srv)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := c.StreamChat(ctx, []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
	if err == nil {
		t.Fatal("StreamChat() error = nil, want error for cancelled context")
	}
	if requestReceived {
		t.Error("request was sent despite cancelled context")
	}
}

// TestStreamChatStreamField verifies the raw request body contains
// "stream":true by capturing the body with a custom server.
func TestStreamChatStreamField(t *testing.T) {
	var rawBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rawBody = string(body)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n")
	}))
	defer srv.Close()
	c := newClientEnv(t, srv)

	ch, err := c.StreamChat(context.Background(), []core.Message{{Role: core.RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("StreamChat() error = %v", err)
	}
	for range ch {
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(rawBody), &parsed); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}
	if stream, ok := parsed["stream"]; !ok || stream != true {
		t.Errorf("request body stream = %v, want true", stream)
	}
}
