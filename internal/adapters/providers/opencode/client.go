// Package opencode implements the OpenCode provider adapter. It reuses the
// openai.Client with a different base URL and API key environment variable
// (REQ-SEL-1). The adapter is thin — zero new HTTP or protocol code.
package opencode

import (
	"errors"
	"os"

	"github.com/biggs-100/kui/internal/adapters/providers/openai"
	"github.com/biggs-100/kui/internal/core"
)

const (
	// defaultBaseURL is the OpenCode Zen endpoint when OPENCODE_BASE_URL is unset.
	defaultBaseURL = "https://opencode.ai/zen/go/v1"
)

// NewClient creates an OpenCode provider from the environment. It reads
// OPENCODE_API_KEY (required) and OPENCODE_BASE_URL (optional, defaults to
// https://opencode.ai/zen/go/v1) and delegates to openai.NewClientWithConfig.
func NewClient(baseURL string) (core.Provider, error) {
	key := os.Getenv("OPENCODE_API_KEY")
	if key == "" {
		return nil, errors.New("OPENCODE_API_KEY is not set: export OPENCODE_API_KEY before running kui")
	}
	if baseURL == "" {
		baseURL = os.Getenv("OPENCODE_BASE_URL")
		if baseURL == "" {
			baseURL = defaultBaseURL
		}
	}
	return openai.NewClientWithConfig(key, baseURL, "")
}
