package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepTool searches file contents using a regexp pattern inside the workspace root.
type GrepTool struct {
	root string
}

// NewGrepTool returns a grep tool confined to root.
func NewGrepTool(root string) *GrepTool {
	return &GrepTool{root: root}
}

// Name returns the stable tool name (REQ-TOOLS-6).
func (t *GrepTool) Name() string { return "grep" }

// Description returns the tool description.
func (t *GrepTool) Description() string {
	return "Search file contents using a regexp pattern inside the workspace"
}

// Schema returns the raw JSON parameter schema (REQ-TOOLS-6).
func (t *GrepTool) Schema() string {
	return `{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string"},"include":{"type":"string"},"max_results":{"type":"integer"}},"required":["pattern"]}`
}

// grepFormat formats a match as "file:line:text".
func grepFormat(file string, line int, text string) string {
	return fmt.Sprintf("%s:%d:%s", file, line, text)
}

// Execute searches files matching pattern. Paths escaping the workspace root
// are rejected before any I/O (REQ-TOOLS-5).
func (t *GrepTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Pattern    string `json:"pattern"`
		Path       string `json:"path"`
		Include    string `json:"include"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.Pattern == "" {
		return "", fmt.Errorf("pattern must not be empty")
	}

	re, err := regexp.Compile(in.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regexp pattern: %w", err)
	}

	maxResults := in.MaxResults
	if maxResults <= 0 {
		maxResults = 100
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

	results, err := grepWalk(root, re, in.Include, maxResults)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(results)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// grepWalk walks root and returns matching lines up to maxResults.
func grepWalk(root string, re *regexp.Regexp, include string, maxResults int) ([]string, error) {
	var results []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip .git directory.
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		// Skip directories.
		if d.IsDir() {
			return nil
		}

		// Apply include filter.
		if include != "" {
			matched, err := filepath.Match(include, d.Name())
			if err != nil || !matched {
				return nil
			}
		}

		// Binary detection: skip files with null bytes in first 512 bytes.
		if isBinary(path) {
			return nil
		}

		// Search line by line.
		matches, err := searchFile(path, root, re)
		if err != nil {
			return err
		}
		results = append(results, matches...)

		if len(results) >= maxResults {
			return io.EOF // stop walking
		}
		return nil
	})

	if err != nil && err != io.EOF {
		return nil, err
	}

	// Truncate to maxResults.
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, nil
}

// searchFile reads path line by line and returns matches as "file:line:text".
func searchFile(path, root string, re *regexp.Regexp) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	// Normalize to forward slashes.
	rel = filepath.ToSlash(rel)

	var matches []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, grepFormat(rel, lineNum, strings.TrimSpace(line)))
		}
	}
	return matches, scanner.Err()
}

// isBinary checks if a file is binary by looking for null bytes in the first 512 bytes.
func isBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true // treat unreadable as binary
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return true
	}

	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}
