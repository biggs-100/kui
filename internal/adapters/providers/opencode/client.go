// Package opencode implements the OpenCode provider adapter. It reuses the
// openai.Client with a different base URL and API key environment variable
// (REQ-SEL-1). The adapter is thin — zero new HTTP or protocol code.
package opencode

import (
	"fmt"

	"github.com/biggs-100/kui/internal/adapters/providers/openai"
	"github.com/biggs-100/kui/internal/core"
)

// NewClient creates an OpenCode provider from the resolved API key and base URL.
// The caller (resolver layer) is responsible for layered API key resolution.
func NewClient(apiKey, baseURL string) (core.Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("opencode API key is empty")
	}
	if baseURL == "" {
		baseURL = "https://opencode.ai/zen/v1"
	}
	return openai.NewClientWithConfig(apiKey, baseURL, "")
}
