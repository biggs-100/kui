package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GlobTool finds files matching a glob pattern inside the workspace root.
type GlobTool struct {
	root string
}

// NewGlobTool returns a glob tool confined to root.
func NewGlobTool(root string) *GlobTool {
	return &GlobTool{root: root}
}

// Name returns the stable tool name (REQ-TOOLS-4).
func (t *GlobTool) Name() string { return "glob" }

// Description returns the tool description (REQ-TOOLS-4).
func (t *GlobTool) Description() string {
	return "Find files matching a glob pattern inside the workspace"
}

// Schema returns the raw JSON parameter schema (D3, REQ-TOOLS-4).
func (t *GlobTool) Schema() string {
	return `{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string"}},"required":["pattern"]}`
}

// Execute finds files matching pattern. Paths escaping the workspace root are
// rejected before any I/O (REQ-TOOLS-5).
func (t *GlobTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.Pattern == "" {
		return "", fmt.Errorf("pattern must not be empty")
	}

	// Resolve the base directory — path is optional (defaults to root).
	root := t.root
	if in.Path != "" {
		resolved, err := resolvePath(root, in.Path)
		if err != nil {
			return "", err
		}
		root = resolved
	}

	matches, err := globWalk(root, in.Pattern)
	if err != nil {
		return "", err
	}

	sort.Strings(matches)
	data, err := json.Marshal(matches)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// globWalk walks root and returns relative paths matching pattern.
// It supports ** for recursive matching. The .git directory is always skipped.
func globWalk(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip .git directory.
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		// Skip directories — only match files.
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if matchGlob(pattern, rel) {
			// Normalize to forward slashes for consistent output.
			matches = append(matches, filepath.ToSlash(rel))
		}
		return nil
	})
	return matches, err
}

// matchGlob checks if relPath matches the glob pattern.
// It supports ** for recursive directory matching.
func matchGlob(pattern, relPath string) bool {
	// Split pattern and path into segments.
	patternSegs := strings.Split(filepath.ToSlash(pattern), "/")
	pathSegs := strings.Split(filepath.ToSlash(relPath), "/")

	return matchSegments(patternSegs, pathSegs)
}

// matchSegments recursively matches pattern segments against path segments.
func matchSegments(pattern, path []string) bool {
	// Both exhausted — match.
	if len(pattern) == 0 && len(path) == 0 {
		return true
	}
	// Pattern exhausted but path remains — no match.
	if len(pattern) == 0 {
		return false
	}

	// Handle ** (recursive wildcard).
	if pattern[0] == "**" {
		// ** can match zero or more path segments.
		for i := 0; i <= len(path); i++ {
			if matchSegments(pattern[1:], path[i:]) {
				return true
			}
		}
		return false
	}

	// Path exhausted but pattern remains (and pattern is not **) — no match.
	if len(path) == 0 {
		return false
	}

	// Match a single segment with filepath.Match.
	matched, err := filepath.Match(pattern[0], path[0])
	if err != nil || !matched {
		return false
	}

	return matchSegments(pattern[1:], path[1:])
}
