package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// defaultWebFetchTimeout is the default HTTP client timeout.
const defaultWebFetchTimeout = 30 * time.Second

// WebFetchTool retrieves a web page via HTTP GET and returns the body text.
type WebFetchTool struct {
	client *http.Client
}

// NewWebFetchTool returns a web fetch tool with the given timeout. A zero
// timeout selects the default of 30s.
func NewWebFetchTool(timeout time.Duration) *WebFetchTool {
	if timeout <= 0 {
		timeout = defaultWebFetchTimeout
	}
	return &WebFetchTool{
		client: &http.Client{Timeout: timeout},
	}
}

// Name returns the stable tool name.
func (t *WebFetchTool) Name() string { return "web_fetch" }

// Description returns the tool description.
func (t *WebFetchTool) Description() string {
	return "Fetch a web page via HTTP GET and return its body text"
}

// Schema returns the raw JSON parameter schema (REQ-TOOLS-7).
func (t *WebFetchTool) Schema() string {
	return `{"type":"object","properties":{"url":{"type":"string"},"format":{"type":"string"}},"required":["url"]}`
}

// Execute fetches the given URL. Only http and https schemes are allowed.
func (t *WebFetchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		URL    string `json:"url"`
		Format string `json:"format"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.URL == "" {
		return "", fmt.Errorf("url must not be empty")
	}

	parsed, err := url.Parse(in.URL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", parsed.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, in.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	content := string(body)

	switch in.Format {
	case "html":
		return content, nil
	case "markdown":
		return stripHTML(content), nil
	default: // "text" or unset
		return content, nil
	}
}

// htmlTagRe matches any HTML tag for stripping.
var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// stripHTML removes HTML tags, collapses whitespace, and returns plain text.
func stripHTML(html string) string {
	text := htmlTagRe.ReplaceAllString(html, " ")
	// Collapse runs of whitespace (spaces, tabs, newlines) into a single space.
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
