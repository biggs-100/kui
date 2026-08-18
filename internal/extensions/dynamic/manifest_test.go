package dynamic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifestValid(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: notes
version: "1.0"
protocol_version: kui-ext/1
entry_point: /usr/bin/notes-ext
`)
	m, err := LoadManifest(filepath.Join(dir, "extension.yaml"))
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if m.Name != "notes" {
		t.Errorf("Name = %q, want %q", m.Name, "notes")
	}
	if m.Version != "1.0" {
		t.Errorf("Version = %q, want %q", m.Version, "1.0")
	}
	if m.ProtocolVersion != "kui-ext/1" {
		t.Errorf("ProtocolVersion = %q, want %q", m.ProtocolVersion, "kui-ext/1")
	}
	if m.EntryPoint != "/usr/bin/notes-ext" {
		t.Errorf("EntryPoint = %q, want %q", m.EntryPoint, "/usr/bin/notes-ext")
	}
}

func TestLoadManifestMissingFile(t *testing.T) {
	_, err := LoadManifest("/nonexistent/path/extension.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadManifestMissingFields(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: notes
`)
	_, err := LoadManifest(filepath.Join(dir, "extension.yaml"))
	if err == nil {
		t.Fatal("expected error for missing required fields, got nil")
	}
}

func TestLoadManifestMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "extension.yaml"), []byte("not: valid: yaml: [[["), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadManifest(filepath.Join(dir, "extension.yaml"))
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestLoadManifestEmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "extension.yaml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadManifest(filepath.Join(dir, "extension.yaml"))
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

func writeManifest(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "extension.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
