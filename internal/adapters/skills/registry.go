package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RegistryClient fetches skill manifests and files from a remote registry.
type RegistryClient struct {
	httpClient *http.Client
	timeout    time.Duration
}

// IndexSkill is one entry in a registry's index.json.
type IndexSkill struct {
	Name    string   `json:"name"`
	Version string   `json:"version,omitempty"`
	Files   []string `json:"files,omitempty"`
}

// RegistryIndex is the parsed index.json response from a registry.
type RegistryIndex struct {
	Skills []IndexSkill `json:"skills"`
}

// NewRegistryClient creates a RegistryClient with the given per-request
// timeout in seconds.
func NewRegistryClient(timeoutSec int) *RegistryClient {
	return &RegistryClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
		timeout: time.Duration(timeoutSec) * time.Second,
	}
}

// FetchIndex downloads and parses index.json from the given base URL.
// The base URL should NOT end with a trailing slash.
func (c *RegistryClient) FetchIndex(ctx context.Context, baseURL string) (*RegistryIndex, error) {
	url := baseURL + "/index.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching index: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading index body: %w", err)
	}

	var index RegistryIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("parsing index: %w", err)
	}
	return &index, nil
}

// FetchFile downloads a single file for the given skill from the registry.
// The URL pattern is {baseURL}/{skillName}/{filename}.
func (c *RegistryClient) FetchFile(ctx context.Context, baseURL, skillName, filename string) ([]byte, error) {
	url := baseURL + "/" + skillName + "/" + filename
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching file: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading file body: %w", err)
	}
	return body, nil
}
