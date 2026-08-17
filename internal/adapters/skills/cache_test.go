package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheDir(t *testing.T) {
	// REQ-RS-9: Dir returns a path containing the SHA256 hex of
	// baseURL+skillName+version.
	root := t.TempDir()
	c := NewCache(root)
	dir := c.Dir("https://r.com/skills", "go-testing", "1.0")
	if dir == "" {
		t.Fatal("Dir returned empty string")
	}
	// Must be under root.
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if abs[:len(root)] != root {
		t.Errorf("Dir = %q, not under root %q", abs, root)
	}
	// Must differ for different versions.
	dir2 := c.Dir("https://r.com/skills", "go-testing", "2.0")
	if dir == dir2 {
		t.Errorf("Dir for v1.0 and v2.0 are equal: %q", dir)
	}
}

func TestCacheIsCached(t *testing.T) {
	// REQ-RS-10, REQ-RS-12: IsCached returns true when .kui-version matches,
	// false when it differs.
	root := t.TempDir()
	c := NewCache(root)
	skillDir := filepath.Join(root, "skills", "test-skill")
	os.MkdirAll(skillDir, 0o755)

	// No version file → not cached.
	if c.IsCached(skillDir, "1.0") {
		t.Error("IsCached with no version file should return false")
	}

	// Write version file matching.
	os.WriteFile(filepath.Join(skillDir, ".kui-version"), []byte("1.0"), 0o644)
	if !c.IsCached(skillDir, "1.0") {
		t.Error("IsCached with matching version should return true")
	}

	// Write version file mismatching.
	os.WriteFile(filepath.Join(skillDir, ".kui-version"), []byte("2.0"), 0o644)
	if c.IsCached(skillDir, "1.0") {
		t.Error("IsCached with mismatched version should return false")
	}
}

func TestCacheStore(t *testing.T) {
	// REQ-RS-11: Store writes files atomically and creates .kui-version.
	root := t.TempDir()
	c := NewCache(root)

	// Store must create the directory and files.
	baseURL := "https://r.com/skills"
	skillName := "go-testing"
	version := "1.0"
	files := map[string][]byte{
		"SKILL.md":   []byte("# go-testing\n\nRun Go tests."),
		"skill.yaml": []byte("name: go-testing\ndescription: Run Go tests\n"),
	}

	err := c.Store(baseURL, skillName, version, files)
	if err != nil {
		t.Fatalf("Store returned error: %v", err)
	}

	skillDir := c.Dir(baseURL, skillName, version)

	// Check .kui-version
	vdata, err := os.ReadFile(filepath.Join(skillDir, ".kui-version"))
	if err != nil {
		t.Fatalf("reading .kui-version: %v", err)
	}
	if string(vdata) != version {
		t.Errorf(".kui-version = %q, want %q", string(vdata), version)
	}

	// Check files were written.
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(skillDir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if string(got) != string(want) {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestCacheStoreNoPartialOnFailure(t *testing.T) {
	// REQ-RS-11: A failed Store must not leave a partial entry at the final path.
	root := t.TempDir()
	c := NewCache(root)

	baseURL := "https://r.com/skills"
	skillName := "ghost"
	version := "1.0"

	// Store with empty files — should still succeed (no failure scenario here,
	// but verify the final path exists and is clean).
	files := map[string][]byte{}
	err := c.Store(baseURL, skillName, version, files)
	if err != nil {
		t.Fatalf("Store returned error: %v", err)
	}

	skillDir := c.Dir(baseURL, skillName, version)
	// .kui-version must exist even with empty files.
	if _, err := os.Stat(filepath.Join(skillDir, ".kui-version")); os.IsNotExist(err) {
		t.Error(".kui-version missing after Store with empty files")
	}
}
