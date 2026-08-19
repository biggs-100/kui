package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Task 2.3 RED: TestGrepValidRegex ────────────────────────────────────────

func TestGrepValidRegex(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "main.go", "package main\n\nfunc main() {\n}\n")

	tool := NewGrepTool(root)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"pattern":"func\\s+\\w+","include":"*.go"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	var results []string
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1: %v", len(results), results)
	}

	want := "main.go:3:func main() {"
	if results[0] != want {
		t.Errorf("results[0] = %q, want %q", results[0], want)
	}
}

// ── Task 2.5 RED: TestGrepNoMatches ─────────────────────────────────────────

func TestGrepNoMatches(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "main.go", "package main\n\nfunc main() {\n}\n")

	tool := NewGrepTool(root)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"pattern":"TODO","include":"*.go"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	var results []string
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("got %d results, want 0: %v", len(results), results)
	}
}

// ── Task 2.6 RED: TestGrepBinarySkipped ──────────────────────────────────────

func TestGrepBinarySkipped(t *testing.T) {
	root := t.TempDir()

	// Create a Go file with a match.
	writeFile(t, root, "main.go", "package main\n")

	// Create a binary file (PNG header + null bytes).
	binPath := filepath.Join(root, "image.png")
	binContent := append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a}, make([]byte, 512)...)
	if err := os.WriteFile(binPath, binContent, 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewGrepTool(root)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"pattern":".*"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	var results []string
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// Only main.go should appear — image.png is binary and must be excluded.
	for _, r := range results {
		if strings.HasPrefix(r, "image.png") {
			t.Errorf("binary file image.png should be excluded, got result: %q", r)
		}
	}
}

// ── Task 2.7 RED: TestGrepMaxResultsCap ──────────────────────────────────────

func TestGrepMaxResultsCap(t *testing.T) {
	root := t.TempDir()

	// Generate 200 matching lines across files.
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString("match here\n")
	}
	writeFile(t, root, "big.go", sb.String())

	tool := NewGrepTool(root)
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"pattern":"match"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	var results []string
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(results) > 100 {
		t.Errorf("got %d results, want at most 100", len(results))
	}
}

// ── Task 2.8 RED: TestGrepPathEscapeRejected ────────────────────────────────

func TestGrepPathEscapeRejected(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "main.go", "package main\n")

	tool := NewGrepTool(root)
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"pattern":"main","path":"../../secret"}`))
	if err == nil {
		t.Fatal("expected error for path escaping workspace root, got nil")
	}

	var pathErr *PathConstraintError
	if !errors.As(err, &pathErr) {
		t.Errorf("expected PathConstraintError, got %T: %v", err, err)
	}
}
