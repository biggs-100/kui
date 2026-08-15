// Package tools provides the built-in agent tools: read_file, write_file,
// and bash. They implement the core.Tool port and confine all file access to
// a workspace root (D11).
package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathConstraintError rejects a path that resolves outside the workspace
// root. The constraint is checked before any I/O happens (D11, REQ-TOOLS-1/2).
type PathConstraintError struct {
	Path string
}

func (e *PathConstraintError) Error() string {
	return fmt.Sprintf("path %q resolves outside the workspace root", e.Path)
}

// resolvePath resolves p against root and verifies the result stays inside
// root. It is symlink-aware: every existing ancestor is evaluated with
// EvalSymlinks so a link pointing outside the root is rejected. Callers must
// run this check before any file I/O.
func resolvePath(root, p string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rootAbs, err = filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", err
	}

	var candidate string
	if filepath.IsAbs(p) {
		candidate = filepath.Clean(p)
	} else {
		candidate = filepath.Join(rootAbs, p)
	}

	resolved, err := evalResolve(candidate)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(rootAbs, resolved)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", &PathConstraintError{Path: p}
	}
	return resolved, nil
}

// evalResolve resolves candidate and every existing ancestor via
// EvalSymlinks. When the final path does not exist yet (the write_file create
// case), the deepest existing ancestor is resolved and the missing tail is
// re-joined.
func evalResolve(candidate string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(candidate); err == nil {
		return resolved, nil
	}
	missing := candidate
	for {
		parent := filepath.Dir(missing)
		if parent == missing {
			return "", fmt.Errorf("no existing ancestor for %q", candidate)
		}
		if resolved, err := filepath.EvalSymlinks(parent); err == nil {
			rel, err := filepath.Rel(parent, candidate)
			if err != nil {
				return "", err
			}
			return filepath.Join(resolved, rel), nil
		}
		missing = parent
	}
}
