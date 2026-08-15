package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolvePathInside resolves relative, nested, and absolute paths that
// stay inside the workspace root (D11, REQ-TOOLS-1/2).
func TestResolvePathInside(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "relative file", path: "notes.md", want: filepath.Join(root, "notes.md")},
		{name: "relative nested file", path: filepath.Join("sub", "notes.md"), want: filepath.Join(root, "sub", "notes.md")},
		{name: "absolute inside root", path: filepath.Join(root, "notes.md"), want: filepath.Join(root, "notes.md")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := resolvePath(root, tt.path)
			if err != nil {
				t.Fatalf("resolvePath(%q) returned error: %v", tt.path, err)
			}
			if resolved != tt.want {
				t.Errorf("resolvePath(%q) = %q, want %q", tt.path, resolved, tt.want)
			}
		})
	}
}

// TestResolvePathEscapeRejected rejects relative escapes, cleaned escapes,
// and absolute paths outside the root with a PathConstraintError.
func TestResolvePathEscapeRejected(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "kui-escape-secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })

	escapes := []string{
		"../kui-escape-secret.txt",
		filepath.Join("a", "..", "..", "kui-escape-secret.txt"),
		outside,
	}
	for _, p := range escapes {
		t.Run(p, func(t *testing.T) {
			_, err := resolvePath(root, p)
			var constraint *PathConstraintError
			if !errors.As(err, &constraint) {
				t.Fatalf("resolvePath(%q) error = %v, want *PathConstraintError", p, err)
			}
			if constraint.Path != p {
				t.Errorf("PathConstraintError.Path = %q, want the original path %q", constraint.Path, p)
			}
		})
	}
}

// TestResolvePathSymlinkEscape rejects paths that reach outside the root
// through a symlink (D11: symlink-safe constraint).
func TestResolvePathSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	_, err := resolvePath(root, filepath.Join("link", "secret.txt"))
	var constraint *PathConstraintError
	if !errors.As(err, &constraint) {
		t.Fatalf("resolvePath via symlink error = %v, want *PathConstraintError", err)
	}
}

// TestResolvePathMissingTailStaysInside resolves a path whose final elements
// do not exist yet (write_file create case) by resolving the deepest existing
// ancestor and re-joining the tail.
func TestResolvePathMissingTailStaysInside(t *testing.T) {
	root := t.TempDir()
	resolved, err := resolvePath(root, filepath.Join("a", "b", "new.txt"))
	if err != nil {
		t.Fatalf("resolvePath with missing parents returned error: %v", err)
	}
	want := filepath.Join(root, "a", "b", "new.txt")
	if resolved != want {
		t.Errorf("resolved = %q, want %q", resolved, want)
	}
	if !strings.HasPrefix(resolved, root) {
		t.Errorf("resolved path %q escapes root %q", resolved, root)
	}
}
