package plugin

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// PluginType represents the type of plugin.
type PluginType string

const (
	PluginTool    PluginType = "tool"
	PluginHook    PluginType = "hook"
	PluginCommand PluginType = "command"
	PluginTheme   PluginType = "theme"
	PluginSkill   PluginType = "skill"
)

// validPluginTypes contains all recognized plugin types for validation.
var validPluginTypes = map[PluginType]bool{
	PluginTool:    true,
	PluginHook:    true,
	PluginCommand: true,
	PluginTheme:   true,
	PluginSkill:   true,
}

// semverPattern validates semantic version strings (MAJOR.MINOR.PATCH).
var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*)?(\+[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*)?$`)

// PluginManifest represents the contents of a kui-plugin.yaml file.
type PluginManifest struct {
	Name            string     `yaml:"name"`
	Version         string     `yaml:"version"`
	Type            PluginType `yaml:"type"`
	EntryPoint      string     `yaml:"entry_point"`
	Description     string     `yaml:"description,omitempty"`
	Capabilities    []string   `yaml:"capabilities,omitempty"`
	Permissions     []string   `yaml:"permissions,omitempty"`
	ProtocolVersion string     `yaml:"protocol_version,omitempty"`
}

// ManifestError is defined in errors.go

// ParseManifest parses YAML bytes into a PluginManifest and validates it.
func ParseManifest(data []byte) (*PluginManifest, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty manifest data")
	}

	var m PluginManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	if err := ValidateManifest(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

// ValidateManifest checks that all required fields are present and valid.
func ValidateManifest(m *PluginManifest) error {
	if m == nil {
		return fmt.Errorf("nil manifest")
	}
	if m.Name == "" {
		return fmt.Errorf("required field name is missing")
	}
	if m.Version == "" {
		return fmt.Errorf("required field version is missing")
	}
	if !semverPattern.MatchString(m.Version) {
		return fmt.Errorf("invalid semver version %q", m.Version)
	}
	if m.Type == "" {
		return fmt.Errorf("required field type is missing")
	}
	if !validPluginTypes[m.Type] {
		return fmt.Errorf("unknown plugin type %q", m.Type)
	}
	if m.EntryPoint == "" {
		return fmt.Errorf("required field entry_point is missing")
	}
	return nil
}

// LoadManifest reads a kui-plugin.yaml file from disk, parses, and validates it.
func LoadManifest(path string) (*PluginManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &ManifestError{File: path, Err: fmt.Errorf("read: %w", err)}
	}

	if len(data) == 0 {
		return nil, &ManifestError{File: path, Err: fmt.Errorf("empty manifest")}
	}

	var m PluginManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, &ManifestError{File: path, Err: fmt.Errorf("parse: %w", err)}
	}

	if err := ValidateManifest(&m); err != nil {
		return nil, &ManifestError{File: path, Err: err}
	}

	return &m, nil
}

// ParseExtensionYAML parses a legacy extension.yaml file into a PluginManifest.
// Legacy manifests are treated as type "tool" with no capabilities or permissions.
func ParseExtensionYAML(data []byte) (*PluginManifest, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty extension data")
	}

	// Parse as a generic map to extract known fields
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	m := &PluginManifest{
		Type: PluginTool, // Legacy defaults to tool
	}

	if v, ok := raw["name"].(string); ok {
		m.Name = v
	}
	if v, ok := raw["version"].(string); ok {
		m.Version = v
	}
	if v, ok := raw["entry_point"].(string); ok {
		m.EntryPoint = v
	}
	if v, ok := raw["protocol_version"].(string); ok {
		m.ProtocolVersion = v
	}
	if v, ok := raw["description"].(string); ok {
		m.Description = v
	}

	// Validate required fields
	if m.Name == "" {
		return nil, fmt.Errorf("required field name is missing")
	}
	if m.Version == "" {
		return nil, fmt.Errorf("required field version is missing")
	}
	if m.EntryPoint == "" {
		return nil, fmt.Errorf("required field entry_point is missing")
	}

	// Normalize version format for legacy manifests (e.g., "1.0" -> "1.0.0")
	m.Version = normalizeVersion(m.Version)

	return m, nil
}

// normalizeVersion ensures a version string has at least major.minor.patch format.
func normalizeVersion(v string) string {
	parts := strings.Split(v, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	return strings.Join(parts[:3], ".")
}
