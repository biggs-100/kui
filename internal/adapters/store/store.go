// Package store implements the .kui persistence adapter (D20, REQ-PROFILE-4).
// It persists the per-profile model memory (models.json) and the active
// profile (active, plain text) under a hidden .kui directory, honoring the
// KUI_HOME environment variable for hermetic tests (D18). All filesystem work
// lives in this adapter — never in the core (REQ-PROFILE-5).
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/biggs-100/kui/internal/core"
)

// dirName is the hidden config directory created under the store root.
const dirName = ".kui"

// modelsFileName holds the per-profile model overrides (map[profile]model).
const modelsFileName = "models.json"

// activeFileName holds the name of the active profile (plain text, D18).
const activeFileName = "active"

// Store persists profile runtime state under .kui (REQ-PROFILE-4, D20). It
// implements the core.ModelMemory port.
type Store struct {
	root string
}

// New creates a store rooted at the given directory. Passing an empty root
// resolves the root from KUI_HOME, falling back to the platform config
// directory (os.UserConfigDir()/kui). KUI_HOME lets tests keep all state in a
// temp directory (D18).
func New(root string) *Store {
	if root == "" {
		if home := os.Getenv("KUI_HOME"); home != "" {
			root = home
		} else if dir, err := os.UserConfigDir(); err == nil {
			root = filepath.Join(dir, "kui")
		} else {
			root = "."
		}
	}
	return &Store{root: root}
}

// Get implements core.ModelMemory (REQ-PROFILE-4): it returns the saved model
// for the profile and whether one exists. An absent store file is no saved
// model; an unreadable store degrades to "no saved model" so the resolution
// chain falls back to the layered config (the port carries no error).
func (s *Store) Get(profile string) (string, bool) {
	models, err := s.readModels()
	if err != nil {
		return "", false
	}
	model, ok := models[profile]
	return model, ok
}

// Set implements core.ModelMemory (REQ-PROFILE-4): it saves the model for the
// profile, preserving every other profile. Failures are wrapped in a typed
// StoreError.
func (s *Store) Set(profile, model string) error {
	models, err := s.readModels()
	if err != nil {
		return err
	}
	models[profile] = model
	return s.writeModels(models)
}

// Active returns the saved active profile name, or "" when none has been
// saved (D18, session-start activation).
func (s *Store) Active() (string, error) {
	data, err := os.ReadFile(s.activePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", &core.StoreError{Op: "read active", Err: err}
	}
	return strings.TrimSpace(string(data)), nil
}

// SetActive persists the active profile name as plain text under .kui/active
// (D18).
func (s *Store) SetActive(profile string) error {
	if err := os.MkdirAll(s.dir(), 0o755); err != nil {
		return &core.StoreError{Op: "mkdir .kui", Err: err}
	}
	if err := os.WriteFile(s.activePath(), []byte(profile), 0o644); err != nil {
		return &core.StoreError{Op: "write active", Err: err}
	}
	return nil
}

func (s *Store) dir() string {
	return filepath.Join(s.root, dirName)
}

func (s *Store) modelsPath() string {
	return filepath.Join(s.dir(), modelsFileName)
}

func (s *Store) activePath() string {
	return filepath.Join(s.dir(), activeFileName)
}

// readModels loads the saved per-profile models. An absent file yields an
// empty map.
func (s *Store) readModels() (map[string]string, error) {
	data, err := os.ReadFile(s.modelsPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, &core.StoreError{Op: "read models", Err: err}
	}
	models := map[string]string{}
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, &core.StoreError{Op: "parse models", Err: err}
	}
	return models, nil
}

// writeModels persists the full model map as JSON (keys sorted by encoding/
// json, so output is stable).
func (s *Store) writeModels(models map[string]string) error {
	data, err := json.Marshal(models)
	if err != nil {
		return &core.StoreError{Op: "encode models", Err: err}
	}
	if err := os.MkdirAll(s.dir(), 0o755); err != nil {
		return &core.StoreError{Op: "mkdir .kui", Err: err}
	}
	if err := os.WriteFile(s.modelsPath(), data, 0o644); err != nil {
		return &core.StoreError{Op: "write models", Err: err}
	}
	return nil
}
