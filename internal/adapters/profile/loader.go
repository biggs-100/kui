// Package profile implements the yaml.v3 profile loader and pure layered
// resolver (REQ-PROFILE-1/2). All profile YAML parsing and filesystem work
// happens in this adapter — never in the core, which stays stdlib-only
// (REQ-PROFILE-5, guard test).
package profile

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/biggs-100/kui/internal/core"
	"gopkg.in/yaml.v3"
)

// profileFileName is the config file resolved in every layer directory.
const profileFileName = "profile.yaml"

// Config is one layer's profile.yaml content (REQ-PROFILE-1).
type Config struct {
	Name         string   `yaml:"name"`
	Model        string   `yaml:"model"`
	Provider     string   `yaml:"provider"`
	SystemPrompt string   `yaml:"system_prompt"`
	Tools        []string `yaml:"tools"`
	Skills       []string `yaml:"skills"`
	Thinking     string   `yaml:"thinking"`
	Permissions  []Rule   `yaml:"permissions"`
}

// Rule is one permission entry declared in profile.yaml (D15). Action is one
// of allow, ask, deny; ask degrades to deny at evaluation time.
type Rule struct {
	Pattern string `yaml:"pattern"`
	Action  string `yaml:"action"`
}

// Profile is a fully resolved profile: each field comes from the nearest
// layer that declares it (profile → project → global, REQ-PROFILE-2), and
// permissions concatenate global → project → profile so a profile rule wins
// on a conflicting pattern at evaluation time (D15).
type Profile struct {
	Name         string
	Model        string
	Provider     string
	SystemPrompt string // absolute path to the SYSTEM.md body
	Tools        []string
	Skills       []string
	Thinking     string
	Permissions  []Rule
}

// Loader resolves named profiles across the three layers (REQ-PROFILE-2):
// the profile root holds the named profiles (<profileRoot>/<name>/profile.yaml
// plus SYSTEM.md), while the project and global directories provide fallback
// config layers.
type Loader struct {
	profileRoot string
	projectDir  string
	globalDir   string
}

// NewLoader creates a loader over the three layer directories.
func NewLoader(profileRoot, projectDir, globalDir string) *Loader {
	return &Loader{
		profileRoot: profileRoot,
		projectDir:  projectDir,
		globalDir:   globalDir,
	}
}

// Resolve loads and merges the named profile across the three layers
// (REQ-PROFILE-2). A profile with no profile.yaml is an UnknownProfileError
// (REQ-PROFILE-3); malformed yaml is a ProfileActivationError naming the
// offending file (REQ-PROFILE-1).
func (l *Loader) Resolve(name string) (*Profile, error) {
	profileDir := filepath.Join(l.profileRoot, name)
	config, err := l.readConfig(filepath.Join(profileDir, profileFileName), name)
	if err != nil {
		return nil, err
	}
	project, err := l.readOptionalConfig(filepath.Join(l.projectDir, profileFileName), name)
	if err != nil {
		return nil, err
	}
	global, err := l.readOptionalConfig(filepath.Join(l.globalDir, profileFileName), name)
	if err != nil {
		return nil, err
	}
	return resolve(name, profileDir, l.projectDir, l.globalDir, config, project, global), nil
}

// SystemPrompt reads the resolved SYSTEM.md body verbatim (REQ-PROFILE-1). A
// missing or unreadable body returns a typed error naming the file.
func (l *Loader) SystemPrompt(p *Profile) (string, error) {
	body, err := os.ReadFile(p.SystemPrompt)
	if err != nil {
		return "", &core.ProfileActivationError{Name: p.Name, File: p.SystemPrompt, Err: err}
	}
	return string(body), nil
}

// Discover returns the names of the profiles in the profile root, in
// directory order. An absent root yields an empty list (REQ-PCLI-1).
func (l *Loader) Discover() ([]string, error) {
	entries, err := os.ReadDir(l.profileRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(l.profileRoot, entry.Name(), profileFileName)); err == nil {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// readConfig reads and parses a required layer config: a missing file is an
// unknown-profile error and a malformed file is an activation error naming
// the file (REQ-PROFILE-1/3).
func (l *Loader) readConfig(path, name string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &core.UnknownProfileError{Name: name}
		}
		return nil, &core.ProfileActivationError{Name: name, File: path, Err: err}
	}
	config, err := parseConfig(data)
	if err != nil {
		return nil, &core.ProfileActivationError{Name: name, File: path, Err: err}
	}
	return config, nil
}

// readOptionalConfig reads and parses an optional layer config: a missing
// file yields a nil config (the layer contributes nothing), while a malformed
// file still fails naming the file (REQ-PROFILE-1).
func (l *Loader) readOptionalConfig(path, name string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, &core.ProfileActivationError{Name: name, File: path, Err: err}
	}
	config, err := parseConfig(data)
	if err != nil {
		return nil, &core.ProfileActivationError{Name: name, File: path, Err: err}
	}
	return config, nil
}

// parseConfig parses one layer's profile.yaml.
func parseConfig(data []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// resolve merges the three layer configs (nearest first: profile, project,
// global) into a Profile (REQ-PROFILE-2). Scalar and list fields come from
// the nearest layer that declares them; permissions concatenate farthest-first
// (global → project → profile) so a profile rule is evaluated last and wins a
// conflicting pattern (D15). It performs no I/O.
func resolve(name, profileDir, projectDir, globalDir string, config, project, global *Config) *Profile {
	p := &Profile{Name: name}
	layers := []struct {
		dir    string
		config *Config
	}{
		{dir: profileDir, config: config},
		{dir: projectDir, config: project},
		{dir: globalDir, config: global},
	}
	for _, layer := range layers {
		if layer.config == nil {
			continue
		}
		if p.Model == "" {
			p.Model = layer.config.Model
		}
		if p.Provider == "" {
			p.Provider = layer.config.Provider
		}
		if p.SystemPrompt == "" && layer.config.SystemPrompt != "" {
			ref := layer.config.SystemPrompt
			if !filepath.IsAbs(ref) {
				ref = filepath.Join(layer.dir, ref)
			}
			p.SystemPrompt = ref
		}
		if len(p.Tools) == 0 {
			p.Tools = layer.config.Tools
		}
		if len(p.Skills) == 0 {
			p.Skills = layer.config.Skills
		}
		if p.Thinking == "" {
			p.Thinking = layer.config.Thinking
		}
	}
	// Permissions concatenate farthest-first so the profile layer is last
	// (D15: defaults → config → profile, last matching rule wins).
	for i := len(layers) - 1; i >= 0; i-- {
		if layers[i].config != nil {
			p.Permissions = append(p.Permissions, layers[i].config.Permissions...)
		}
	}
	return p
}
