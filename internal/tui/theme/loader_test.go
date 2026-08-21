package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBytes_Valid(t *testing.T) {
	data := []byte(`{"name":"test","bg":"#000000","fg":"#ffffff"}`)
	th, err := ParseBytes(data)
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}
	if th.Name != "test" {
		t.Errorf("Name = %q, want test", th.Name)
	}
}

func TestParseBytes_Invalid(t *testing.T) {
	_, err := ParseBytes([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseFile_Loads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "theme.json")
	data := []byte(`{"name":"custom","bg":"#111111","fg":"#eeeeee"}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	th, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if th.BG != "#111111" {
		t.Errorf("BG = %q, want #111111", th.BG)
	}
}

func TestDiscoverFindsFileThemes(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	// Create themes/custom.json in dir1
	themesDir1 := filepath.Join(dir1, "themes")
	if err := os.MkdirAll(themesDir1, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	data1 := []byte(`{"name":"custom","bg":"#111111","fg":"#aaaaaa","primary":"#ff0000"}`)
	if err := os.WriteFile(filepath.Join(themesDir1, "custom.json"), data1, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	// Discover should find it
	themes := Discover([]string{dir1})
	if th, ok := themes["custom"]; !ok {
		t.Fatal("Discover did not find custom theme")
	} else if th.Primary != "#ff0000" {
		t.Errorf("Primary = %q, want #ff0000", th.Primary)
	}
	// Test override: later dir overrides earlier
	themesDir2 := filepath.Join(dir2, "themes")
	if err := os.MkdirAll(themesDir2, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	data2 := []byte(`{"name":"custom","bg":"#222222","fg":"#bbbbbb","primary":"#00ff00"}`)
	if err := os.WriteFile(filepath.Join(themesDir2, "custom.json"), data2, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	themes = Discover([]string{dir1, dir2})
	if th, ok := themes["custom"]; !ok {
		t.Fatal("Discover did not find custom after override")
	} else if th.Primary != "#00ff00" {
		t.Errorf("override: Primary = %q, want #00ff00 (later dir should win)", th.Primary)
	}
}

func TestDiscoverIgnoresNonJSON(t *testing.T) {
	dir := t.TempDir()
	themesDir := filepath.Join(dir, "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	// Non-json file should be ignored
	if err := os.WriteFile(filepath.Join(themesDir, "readme.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	themes := Discover([]string{dir})
	if _, ok := themes["readme"]; ok {
		t.Error("Discover should ignore non-json files")
	}
}
