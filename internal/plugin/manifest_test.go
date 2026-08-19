package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifestValid(t *testing.T) {
	dir := t.TempDir()
	writePluginManifest(t, dir, `name: my-tool
version: "1.0.0"
type: tool
entry_point: ./bin/my-tool
description: A useful tool
capabilities:
  - tools:read
  - hooks:on_turn_start
permissions:
  - tools:read
  - filesystem:read
protocol_version: kui-ext/1
`)
	path := filepath.Join(dir, "kui-plugin.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	if m.Name != "my-tool" {
		t.Errorf("Name = %q, want %q", m.Name, "my-tool")
	}
	if m.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", m.Version, "1.0.0")
	}
	if m.Type != PluginTool {
		t.Errorf("Type = %q, want %q", m.Type, PluginTool)
	}
	if m.EntryPoint != "./bin/my-tool" {
		t.Errorf("EntryPoint = %q, want %q", m.EntryPoint, "./bin/my-tool")
	}
	if m.Description != "A useful tool" {
		t.Errorf("Description = %q, want %q", m.Description, "A useful tool")
	}
	if len(m.Capabilities) != 2 {
		t.Errorf("Capabilities len = %d, want 2", len(m.Capabilities))
	}
	if len(m.Permissions) != 2 {
		t.Errorf("Permissions len = %d, want 2", len(m.Permissions))
	}
	if m.ProtocolVersion != "kui-ext/1" {
		t.Errorf("ProtocolVersion = %q, want %q", m.ProtocolVersion, "kui-ext/1")
	}
}

func TestParseManifestMinimal(t *testing.T) {
	input := []byte(`name: simple-plugin
version: "0.1.0"
type: hook
entry_point: ./run.sh
`)
	m, err := ParseManifest(input)
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	if m.Name != "simple-plugin" {
		t.Errorf("Name = %q, want %q", m.Name, "simple-plugin")
	}
	if m.Type != PluginHook {
		t.Errorf("Type = %q, want %q", m.Type, PluginHook)
	}
	if m.Description != "" {
		t.Errorf("Description = %q, want empty", m.Description)
	}
	if len(m.Capabilities) != 0 {
		t.Errorf("Capabilities len = %d, want 0", len(m.Capabilities))
	}
	if len(m.Permissions) != 0 {
		t.Errorf("Permissions len = %d, want 0", len(m.Permissions))
	}
}

func TestParseManifestInvalidYAML(t *testing.T) {
	input := []byte(`not: valid: yaml: [[[
`)
	_, err := ParseManifest(input)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestParseManifestEmptyInput(t *testing.T) {
	input := []byte(``)
	_, err := ParseManifest(input)
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

func TestParseManifestMissingName(t *testing.T) {
	input := []byte(`version: "1.0.0"
type: tool
entry_point: ./bin/plugin
`)
	_, err := ParseManifest(input)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestParseManifestMissingVersion(t *testing.T) {
	input := []byte(`name: my-plugin
type: tool
entry_point: ./bin/plugin
`)
	_, err := ParseManifest(input)
	if err == nil {
		t.Fatal("expected error for missing version, got nil")
	}
}

func TestParseManifestMissingType(t *testing.T) {
	input := []byte(`name: my-plugin
version: "1.0.0"
entry_point: ./bin/plugin
`)
	_, err := ParseManifest(input)
	if err == nil {
		t.Fatal("expected error for missing type, got nil")
	}
}

func TestParseManifestMissingEntryPoint(t *testing.T) {
	input := []byte(`name: my-plugin
version: "1.0.0"
type: tool
`)
	_, err := ParseManifest(input)
	if err == nil {
		t.Fatal("expected error for missing entry_point, got nil")
	}
}

func TestValidateManifestValid(t *testing.T) {
	m := &PluginManifest{
		Name:       "test-plugin",
		Version:    "1.0.0",
		Type:       PluginTool,
		EntryPoint: "./bin/plugin",
	}
	if err := ValidateManifest(m); err != nil {
		t.Fatalf("ValidateManifest() error = %v", err)
	}
}

func TestValidateManifestInvalidVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"no-semver", "not-a-version"},
		{"missing-patch", "1.0"},
		{"extra-parts", "1.0.0.0"},
		{"non-numeric", "a.b.c"},
		{"pre-release-only", "1.0.0-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &PluginManifest{
				Name:       "test",
				Version:    tt.version,
				Type:       PluginTool,
				EntryPoint: "./bin/plugin",
			}
			if err := ValidateManifest(m); err == nil {
				t.Errorf("expected error for version %q, got nil", tt.version)
			}
		})
	}
}

func TestValidateManifestMissingFields(t *testing.T) {
	tests := []struct {
		name string
		m    *PluginManifest
	}{
		{
			"empty-name",
			&PluginManifest{Version: "1.0.0", Type: PluginTool, EntryPoint: "./bin/plugin"},
		},
		{
			"empty-version",
			&PluginManifest{Name: "test", Type: PluginTool, EntryPoint: "./bin/plugin"},
		},
		{
			"empty-type",
			&PluginManifest{Name: "test", Version: "1.0.0", EntryPoint: "./bin/plugin"},
		},
		{
			"empty-entry-point",
			&PluginManifest{Name: "test", Version: "1.0.0", Type: PluginTool},
		},
		{
			"nil-manifest",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateManifest(tt.m); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestValidateManifestInvalidType(t *testing.T) {
	m := &PluginManifest{
		Name:       "test",
		Version:    "1.0.0",
		Type:       PluginType("unknown"),
		EntryPoint: "./bin/plugin",
	}
	if err := ValidateManifest(m); err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}

func TestPluginTypes(t *testing.T) {
	expected := []PluginType{PluginTool, PluginHook, PluginCommand, PluginTheme, PluginSkill}
	seen := make(map[PluginType]bool)
	for _, pt := range expected {
		if pt == "" {
			t.Error("PluginType should not be empty")
		}
		seen[pt] = true
	}
	if len(seen) != 5 {
		t.Errorf("expected 5 plugin types, got %d", len(seen))
	}
}

func TestLoadManifestValid(t *testing.T) {
	dir := t.TempDir()
	writePluginManifest(t, dir, `name: loaded-plugin
version: "2.0.0"
type: command
entry_point: ./cmd/run
`)
	m, err := LoadManifest(filepath.Join(dir, "kui-plugin.yaml"))
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if m.Name != "loaded-plugin" {
		t.Errorf("Name = %q, want %q", m.Name, "loaded-plugin")
	}
}

func TestLoadManifestFileNotFound(t *testing.T) {
	_, err := LoadManifest("/nonexistent/path/kui-plugin.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadManifestMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kui-plugin.yaml")
	if err := os.WriteFile(path, []byte("not: valid: yaml: [[["), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestLegacyExtensionYAML(t *testing.T) {
	input := []byte(`name: legacy-extension
version: "1.0.0"
protocol_version: kui-ext/1
entry_point: ./bin/legacy
`)
	m, err := ParseExtensionYAML(input)
	if err != nil {
		t.Fatalf("ParseExtensionYAML() error = %v", err)
	}
	if m.Name != "legacy-extension" {
		t.Errorf("Name = %q, want %q", m.Name, "legacy-extension")
	}
	if m.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", m.Version, "1.0.0")
	}
	if m.Type != PluginTool {
		t.Errorf("Type = %q, want %q (legacy defaults to tool)", m.Type, PluginTool)
	}
	if m.EntryPoint != "./bin/legacy" {
		t.Errorf("EntryPoint = %q, want %q", m.EntryPoint, "./bin/legacy")
	}
	if len(m.Capabilities) != 0 {
		t.Errorf("Capabilities len = %d, want 0 (legacy has none)", len(m.Capabilities))
	}
	if len(m.Permissions) != 0 {
		t.Errorf("Permissions len = %d, want 0 (legacy has none)", len(m.Permissions))
	}
}

func TestLegacyExtensionYAMLInvalid(t *testing.T) {
	input := []byte(`not: valid: yaml: [[[
`)
	_, err := ParseExtensionYAML(input)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func writePluginManifest(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "kui-plugin.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
