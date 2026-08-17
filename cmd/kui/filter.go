package main

import (
	"strings"

	"github.com/biggs-100/kui/internal/core"
)

// filterTools applies --tools, --exclude-tools, and --no-tools to a full tool
// registry. The returned registry is a new copy — the original is never mutated
// (REQ-CLI-18). When both include and exclude are specified, exclude wins
// (REQ-CLI-17).
func filterTools(full *core.Registry, include, exclude string, noTools bool) *core.Registry {
	if noTools {
		return core.NewRegistry()
	}

	includeSet := splitToolNames(include)
	excludeSet := splitToolNames(exclude)

	result := core.NewRegistry()

	for _, tool := range full.List() {
		name := tool.Name()

		// Exclude always wins (REQ-CLI-17).
		if excludeSet[name] {
			continue
		}

		// If include is specified, only keep named tools (REQ-CLI-14).
		if len(includeSet) > 0 && !includeSet[name] {
			continue
		}

		_ = result.Register(tool)
	}

	return result
}

// splitToolNames parses a comma-separated list of tool names into a set.
// Returns nil for empty input so callers can use a simple len() check.
func splitToolNames(s string) map[string]bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := make(map[string]bool)
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			parts[t] = true
		}
	}
	return parts
}
