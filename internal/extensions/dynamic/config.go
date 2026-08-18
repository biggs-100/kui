package dynamic

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the top-level dynamic extension configuration loaded from
// extensions.yaml. It contains additional directories to scan for extensions
// beyond the default global and project extension directories.
type Config struct {
	Paths []string `yaml:"paths"`
}

// ExtensionsConfig is the YAML wrapper matching the extensions.yaml structure.
type ExtensionsConfig struct {
	Extensions Config `yaml:"extensions"`
}

// LoadConfigs reads global and optional project extensions.yaml files, merges
// them (project paths extend global, deduplicating on collision), and returns
// the merged config. Either path may be empty or absent; missing files are
// treated as empty configs.
func LoadConfigs(globalPath, projectPath string) (*Config, error) {
	base, err := readExtensionsConfig(globalPath)
	if err != nil {
		return nil, err
	}

	override, err := readExtensionsConfig(projectPath)
	if err != nil {
		return nil, err
	}

	merged := mergeConfigs(base, override)
	return merged, nil
}

// readExtensionsConfig parses a single extensions.yaml file into a Config.
// Returns an empty Config when the path is empty or the file does not exist.
func readExtensionsConfig(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading extensions config %s: %w", path, err)
	}

	var wrapper ExtensionsConfig
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing extensions config %s: %w", path, err)
	}

	return &wrapper.Extensions, nil
}

// mergeConfigs returns a new Config where override paths extend base paths.
// Duplicate paths are deduplicated — the first occurrence wins (base order
// preserved, project paths appended when new).
func mergeConfigs(base, override *Config) *Config {
	seen := make(map[string]bool, len(base.Paths)+len(override.Paths))
	merged := &Config{
		Paths: make([]string, 0, len(base.Paths)+len(override.Paths)),
	}

	for _, p := range base.Paths {
		if !seen[p] {
			seen[p] = true
			merged.Paths = append(merged.Paths, p)
		}
	}

	for _, p := range override.Paths {
		if !seen[p] {
			seen[p] = true
			merged.Paths = append(merged.Paths, p)
		}
	}

	return merged
}
