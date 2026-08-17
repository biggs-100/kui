package mcp

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds the configuration for a single MCP server entry.
type ServerConfig struct {
	Type        string            `yaml:"type"`
	Command     []string          `yaml:"command"`
	Cwd         string            `yaml:"cwd,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Disabled    bool              `yaml:"disabled,omitempty"`
	Timeout     time.Duration     `yaml:"timeout,omitempty"`
}

// Config is the top-level MCP configuration loaded from mcp.yaml.
type Config struct {
	Servers map[string]ServerConfig `yaml:"servers"`
}

const defaultTimeout = 30 * time.Second

// LoadConfig reads global and optional project YAML files, merges them
// (project overrides global per REQ-MCP-4), and validates the result.
// Either path may be empty or absent; missing files are treated as empty configs.
func LoadConfig(globalPath, projectPath string) (*Config, error) {
	base, err := readConfigFile(globalPath)
	if err != nil {
		return nil, err
	}

	override, err := readConfigFile(projectPath)
	if err != nil {
		return nil, err
	}

	merged := mergeConfigs(base, override)

	if err := validateConfig(merged); err != nil {
		return nil, err
	}

	applyDefaults(merged)
	return merged, nil
}

// readConfigFile parses a single YAML file into a Config. Returns an empty
// Config when the path is empty or the file does not exist.
func readConfigFile(path string) (*Config, error) {
	if path == "" {
		return &Config{Servers: make(map[string]ServerConfig)}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Servers: make(map[string]ServerConfig)}, nil
		}
		return nil, fmt.Errorf("reading mcp config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing mcp config %s: %w", path, err)
	}
	if cfg.Servers == nil {
		cfg.Servers = make(map[string]ServerConfig)
	}
	return &cfg, nil
}

// mergeConfigs returns a new Config where override servers replace base
// servers entirely (per REQ-MCP-4). Servers only in base are preserved as-is.
func mergeConfigs(base, override *Config) *Config {
	merged := &Config{Servers: make(map[string]ServerConfig, len(base.Servers)+len(override.Servers))}

	// Copy all base servers.
	for name, srv := range base.Servers {
		merged.Servers[name] = srv
	}

	// Override replaces base entirely for same-name servers.
	for name, srv := range override.Servers {
		merged.Servers[name] = srv
	}

	return merged
}

// validateConfig checks for type and command field constraints (REQ-MCP-2, REQ-MCP-3).
func validateConfig(cfg *Config) error {
	for name, srv := range cfg.Servers {
		if srv.Type != "local" {
			return &MCPConfigError{
				File:  "mcp.yaml",
				Field: fmt.Sprintf("servers.%s.type", name),
				Err:   fmt.Errorf("unsupported type %q (only \"local\" is supported)", srv.Type),
			}
		}
		if len(srv.Command) == 0 {
			return &MCPConfigError{
				File:  "mcp.yaml",
				Field: fmt.Sprintf("servers.%s.command", name),
				Err:   fmt.Errorf("command is required"),
			}
		}
	}
	return nil
}

// applyDefaults fills in default values for optional fields (REQ-MCP-3).
func applyDefaults(cfg *Config) {
	for name, srv := range cfg.Servers {
		if srv.Timeout == 0 {
			srv.Timeout = defaultTimeout
		}
		cfg.Servers[name] = srv
	}
}
