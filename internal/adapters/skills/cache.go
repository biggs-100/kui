package skills

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// Cache manages the disk cache for remote skills fetched from registries.
type Cache struct {
	root string
}

// NewCache creates a Cache rooted at the given directory.
func NewCache(root string) *Cache {
	return &Cache{root: root}
}

// Dir returns the cache directory for a specific baseURL+skillName+version.
// The directory name is the SHA256 hex of the concatenated key.
func (c *Cache) Dir(baseURL, skillName, version string) string {
	key := baseURL + "/" + skillName + "/" + version
	h := sha256.Sum256([]byte(key))
	return filepath.Join(c.root, fmt.Sprintf("%x", h))
}

// IsCached reports whether the cached version matches the expected version.
func (c *Cache) IsCached(skillDir, version string) bool {
	vdata, err := os.ReadFile(filepath.Join(skillDir, ".kui-version"))
	if err != nil {
		return false
	}
	return string(vdata) == version
}

// Store atomically writes fetched files to the cache directory for the given
// baseURL+skillName+version. It writes to a staging directory first, then
// renames to the final path (REQ-RS-11).
func (c *Cache) Store(baseURL, skillName, version string, files map[string][]byte) error {
	finalDir := c.Dir(baseURL, skillName, version)
	tmpDir := finalDir + ".tmp"

	// Clean up any leftover staging directory.
	os.RemoveAll(tmpDir)

	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return fmt.Errorf("creating staging dir: %w", err)
	}

	// Write all files to staging.
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), data, 0o644); err != nil {
			os.RemoveAll(tmpDir)
			return fmt.Errorf("writing %s: %w", name, err)
		}
	}

	// Write version marker.
	if err := os.WriteFile(filepath.Join(tmpDir, ".kui-version"), []byte(version), 0o644); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("writing .kui-version: %w", err)
	}

	// Remove old final dir if it exists, then rename staging to final.
	os.RemoveAll(finalDir)
	if err := os.Rename(tmpDir, finalDir); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("renaming staging to final: %w", err)
	}

	return nil
}
