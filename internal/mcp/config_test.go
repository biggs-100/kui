package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfigGlobalOnly verifies REQ-MCP-1: loading a global config with
// one server and no project config returns that server.
func TestLoadConfigGlobalOnly(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yaml")
	content := `
servers:
  myserver:
    type: local
    command: ["echo", "hello"]
`
	if err := os.WriteFile(globalPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(globalPath, "")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.Servers) != 1 {
		t.Fatalf("got %d servers, want 1", len(cfg.Servers))
	}
	srv, ok := cfg.Servers["myserver"]
	if !ok {
		t.Fatal("server 'myserver' not found")
	}
	if srv.Type != "local" {
		t.Errorf("Type = %q, want %q", srv.Type, "local")
	}
	if len(srv.Command) != 2 || srv.Command[0] != "echo" || srv.Command[1] != "hello" {
		t.Errorf("Command = %v, want [echo hello]", srv.Command)
	}
}

// TestLoadConfigProjectOverridesGlobal verifies REQ-MCP-4: when both global
// and project define the same server name, the project entry wins entirely.
func TestLoadConfigProjectOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yaml")
	projectPath := filepath.Join(dir, "project.yaml")

	globalContent := `
servers:
  github:
    type: local
    command: ["gh-global"]
`
	projectContent := `
servers:
  github:
    type: local
    command: ["gh-project"]
`
	if err := os.WriteFile(globalPath, []byte(globalContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(globalPath, projectPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	srv, ok := cfg.Servers["github"]
	if !ok {
		t.Fatal("server 'github' not found")
	}
	if len(srv.Command) != 1 || srv.Command[0] != "gh-project" {
		t.Errorf("Command = %v, want [gh-project]", srv.Command)
	}
}

// TestLoadConfigEmpty verifies REQ-MCP-1: when neither config file exists,
// MCP is disabled (empty server list).
func TestLoadConfigEmpty(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "nonexistent-global.yaml")
	projectPath := filepath.Join(dir, "nonexistent-project.yaml")

	cfg, err := LoadConfig(globalPath, projectPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.Servers) != 0 {
		t.Errorf("got %d servers, want 0", len(cfg.Servers))
	}
}

// TestLoadConfigMissingCommand verifies REQ-MCP-3: a server entry without a
// command field produces a config error.
func TestLoadConfigMissingCommand(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "bad.yaml")
	content := `
servers:
  broken:
    type: local
`
	if err := os.WriteFile(globalPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(globalPath, "")
	if err == nil {
		t.Fatal("expected error for missing command, got nil")
	}
}

// TestLoadConfigDisabledServer verifies REQ-MCP-3: disabled servers are
// parsed but flagged.
func TestLoadConfigDisabledServer(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "disabled.yaml")
	content := `
servers:
  offserver:
    type: local
    command: ["echo"]
    disabled: true
`
	if err := os.WriteFile(globalPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(globalPath, "")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	srv, ok := cfg.Servers["offserver"]
	if !ok {
		t.Fatal("server 'offserver' not found")
	}
	if !srv.Disabled {
		t.Error("Disabled = false, want true")
	}
}

// TestLoadConfigDefaults verifies REQ-MCP-3: minimal config gets correct defaults.
func TestLoadConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "minimal.yaml")
	content := `
servers:
  minimal:
    type: local
    command: ["node", "server.js"]
`
	if err := os.WriteFile(globalPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(globalPath, "")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	srv := cfg.Servers["minimal"]
	if srv.Disabled {
		t.Error("Disabled should default to false")
	}
	if srv.Cwd != "" {
		t.Errorf("Cwd = %q, want empty", srv.Cwd)
	}
	if srv.Environment != nil {
		t.Errorf("Environment = %v, want nil", srv.Environment)
	}
}

// TestLoadConfigMergeServersOnlyInGlobal verifies REQ-MCP-4: servers only
// in global are present in merged config.
func TestLoadConfigMergeServersOnlyInGlobal(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yaml")
	projectPath := filepath.Join(dir, "project.yaml")

	globalContent := `
servers:
  a:
    type: local
    command: ["a-cmd"]
  b:
    type: local
    command: ["b-cmd"]
`
	projectContent := `
servers:
  b:
    type: local
    command: ["b-project"]
`
	if err := os.WriteFile(globalPath, []byte(globalContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(globalPath, projectPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	// "a" only in global → present
	if _, ok := cfg.Servers["a"]; !ok {
		t.Error("server 'a' from global should be present")
	}
	// "b" in both → project wins
	if cfg.Servers["b"].Command[0] != "b-project" {
		t.Errorf("server 'b' command = %v, want [b-project]", cfg.Servers["b"].Command)
	}
}

// TestLoadConfigUnknownType verifies REQ-MCP-2: unknown server type is rejected.
func TestLoadConfigUnknownType(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "badtype.yaml")
	content := `
servers:
  remote:
    type: remote
    command: ["echo"]
`
	if err := os.WriteFile(globalPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(globalPath, "")
	if err == nil {
		t.Fatal("expected error for unknown type, got nil")
	}
}
