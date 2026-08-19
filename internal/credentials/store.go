// Package credentials manages provider API keys in .kui/credentials.json
// (REQ-CRED-1 through REQ-CRED-5). It follows the same .kui/ persistence
// pattern used by internal/adapters/store for profile state.
package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// dirName is the hidden config directory under the project root.
	dirName = ".kui"

	// credFileName holds the provider credentials JSON file.
	credFileName = "credentials.json"
)

// credentials is the JSON shape of .kui/credentials.json.
type credentials struct {
	Providers map[string]providerCreds `json:"providers"`
}

// providerCreds holds the API key for a single provider.
type providerCreds struct {
	APIKey string `json:"api_key"`
}

// CredentialStore manages provider API keys persisted as JSON under .kui/.
type CredentialStore struct {
	root  string
	creds credentials
}

// NewCredentialStore creates a store rooted at the given directory. The
// credentials file resolves to root/.kui/credentials.json.
func NewCredentialStore(root string) *CredentialStore {
	return &CredentialStore{root: root}
}

// dir returns the path to the .kui directory.
func (cs *CredentialStore) dir() string {
	return filepath.Join(cs.root, dirName)
}

// credPath returns the absolute path to the credentials file.
func (cs *CredentialStore) credPath() string {
	return filepath.Join(cs.dir(), credFileName)
}

// Load reads and parses the credentials file. If the file does not exist, the
// store remains empty (no error). Malformed JSON returns a descriptive error.
func (cs *CredentialStore) Load() error {
	creds, err := cs.readCreds()
	if err != nil {
		return err
	}
	cs.creds = creds
	return nil
}

// Save writes the current credentials to disk. On Unix the file is created
// with 0600 permissions (owner read/write only). On Windows the permission
// flag is silently ignored.
func (cs *CredentialStore) Save() error {
	return cs.writeCreds(cs.creds)
}

// GetAPIKey returns the stored API key for the given provider. Returns an
// error if the provider has no saved key.
func (cs *CredentialStore) GetAPIKey(provider string) (string, error) {
	if cs.creds.Providers == nil {
		return "", fmt.Errorf("no API key stored for provider %q", provider)
	}
	cred, ok := cs.creds.Providers[provider]
	if !ok || cred.APIKey == "" {
		return "", fmt.Errorf("no API key stored for provider %q", provider)
	}
	return cred.APIKey, nil
}

// SetAPIKey saves the API key for the given provider and persists to disk.
func (cs *CredentialStore) SetAPIKey(provider, key string) error {
	if cs.creds.Providers == nil {
		cs.creds.Providers = map[string]providerCreds{}
	}
	cs.creds.Providers[provider] = providerCreds{APIKey: key}
	return cs.writeCreds(cs.creds)
}

// readCreds loads the credentials from disk. An absent file yields an empty
// credentials struct.
func (cs *CredentialStore) readCreds() (credentials, error) {
	data, err := os.ReadFile(cs.credPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return credentials{}, nil
		}
		return credentials{}, fmt.Errorf("read credentials: %w", err)
	}
	var creds credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return credentials{}, fmt.Errorf("parse credentials: %w", err)
	}
	return creds, nil
}

// writeCreds persists the credentials struct as JSON. It creates the .kui
// directory if needed and uses 0600 permissions on Unix.
func (cs *CredentialStore) writeCreds(creds credentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("encode credentials: %w", err)
	}
	if err := os.MkdirAll(cs.dir(), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dirName, err)
	}
	perm := os.FileMode(0o644)
	if runtime.GOOS != "windows" {
		perm = 0o600
	}
	if err := os.WriteFile(cs.credPath(), data, perm); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}
