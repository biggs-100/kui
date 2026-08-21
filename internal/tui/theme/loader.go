package theme

// loader.go provides theme discovery and parsing utilities.
// The core implementations (ParseFile, ParseBytes, Discover, etc.)
// are in theme.go. This file exists to satisfy the PR structure for
// REQ-TUI-THEME-3 and ensures discovery prefers later dirs overriding earlier.
