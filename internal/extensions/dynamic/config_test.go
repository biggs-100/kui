package dynamic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigsEmpty(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "nonexistent-global.yaml")
	projectPath := filepath.Join(dir, "nonexistent-project.yaml")

	cfg, err := LoadConfigs(globalPath, projectPath)
	if err != nil {
		t.Fatalf("LoadConfigs() error = %v", err)
	}
	if len(cfg.Paths) != 0 {
		t.Errorf("got %d paths, want 0", len(cfg.Paths))
	}
}

func TestLoadConfigsGlobalOnly(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yaml")
	writeExtensionsConfig(t, globalPath, `
extensions:
  paths:
    - /global/ext1
    - /global/ext2
`)

	cfg, err := LoadConfigs(globalPath, "")
	if err != nil {
		t.Fatalf("LoadConfigs() error = %v", err)
	}
	if len(cfg.Paths) != 2 {
		t.Fatalf("got %d paths, want 2", len(cfg.Paths))
	}
	if cfg.Paths[0] != "/global/ext1" {
		t.Errorf("Paths[0] = %q, want %q", cfg.Paths[0], "/global/ext1")
	}
	if cfg.Paths[1] != "/global/ext2" {
		t.Errorf("Paths[1] = %q, want %q", cfg.Paths[1], "/global/ext2")
	}
}

func TestLoadConfigsProjectOnly(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "project.yaml")
	writeExtensionsConfig(t, projectPath, `
extensions:
  paths:
    - /project/ext1
`)

	cfg, err := LoadConfigs("", projectPath)
	if err != nil {
		t.Fatalf("LoadConfigs() error = %v", err)
	}
	if len(cfg.Paths) != 1 {
		t.Fatalf("got %d paths, want 1", len(cfg.Paths))
	}
	if cfg.Paths[0] != "/project/ext1" {
		t.Errorf("Paths[0] = %q, want %q", cfg.Paths[0], "/project/ext1")
	}
}

func TestLoadConfigsNameCollision(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yaml")
	projectPath := filepath.Join(dir, "project.yaml")

	writeExtensionsConfig(t, globalPath, `
extensions:
  paths:
    - /shared/ext
    - /global/ext
`)
	writeExtensionsConfig(t, projectPath, `
extensions:
  paths:
    - /shared/ext
    - /project/ext
`)

	cfg, err := LoadConfigs(globalPath, projectPath)
	if err != nil {
		t.Fatalf("LoadConfigs() error = %v", err)
	}
	// /shared/ext appears in both — deduplicated, project wins
	// /global/ext from global, /project/ext from project
	if len(cfg.Paths) != 3 {
		t.Fatalf("got %d paths, want 3 (deduped /shared/ext)", len(cfg.Paths))
	}
	if cfg.Paths[0] != "/shared/ext" {
		t.Errorf("Paths[0] = %q, want %q", cfg.Paths[0], "/shared/ext")
	}
	if cfg.Paths[1] != "/global/ext" {
		t.Errorf("Paths[1] = %q, want %q", cfg.Paths[1], "/global/ext")
	}
	if cfg.Paths[2] != "/project/ext" {
		t.Errorf("Paths[2] = %q, want %q", cfg.Paths[2], "/project/ext")
	}
}

func TestLoadConfigsPathsExtension(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yaml")
	projectPath := filepath.Join(dir, "project.yaml")

	writeExtensionsConfig(t, globalPath, `
extensions:
  paths:
    - /global/a
`)
	writeExtensionsConfig(t, projectPath, `
extensions:
  paths:
    - /project/b
`)

	cfg, err := LoadConfigs(globalPath, projectPath)
	if err != nil {
		t.Fatalf("LoadConfigs() error = %v", err)
	}
	if len(cfg.Paths) != 2 {
		t.Fatalf("got %d paths, want 2", len(cfg.Paths))
	}
	if cfg.Paths[0] != "/global/a" {
		t.Errorf("Paths[0] = %q, want %q", cfg.Paths[0], "/global/a")
	}
	if cfg.Paths[1] != "/project/b" {
		t.Errorf("Paths[1] = %q, want %q", cfg.Paths[1], "/project/b")
	}
}

func TestLoadConfigsMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(globalPath, []byte("not: valid: yaml: [[["), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfigs(globalPath, "")
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func writeExtensionsConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
