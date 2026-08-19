package lsp

import "sync"

// DiagnosticCache is a thread-safe, push-based cache for LSP diagnostics.
// Diagnostics are keyed by document URI (e.g., "file:///path/to/file.go").
type DiagnosticCache struct {
	mu    sync.RWMutex
	diags map[string][]Diagnostic
}

// NewDiagnosticCache creates a new empty DiagnosticCache.
func NewDiagnosticCache() *DiagnosticCache {
	return &DiagnosticCache{
		diags: make(map[string][]Diagnostic),
	}
}

// Set replaces all diagnostics for a given URI. Thread-safe.
func (dc *DiagnosticCache) Set(uri string, diagnostics []Diagnostic) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.diags[uri] = diagnostics
}

// Get returns the diagnostics for a given URI. Thread-safe.
// Returns nil if no diagnostics exist for the URI.
func (dc *DiagnosticCache) Get(uri string) []Diagnostic {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.diags[uri]
}

// GetWorkspace returns a copy of all diagnostics across all files. Thread-safe.
func (dc *DiagnosticCache) GetWorkspace() map[string][]Diagnostic {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	result := make(map[string][]Diagnostic, len(dc.diags))
	for k, v := range dc.diags {
		result[k] = v
	}
	return result
}

// Clear removes all diagnostics for a given URI. Thread-safe.
func (dc *DiagnosticCache) Clear(uri string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	delete(dc.diags, uri)
}

// ClearAll resets the cache, removing all diagnostics. Thread-safe.
func (dc *DiagnosticCache) ClearAll() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.diags = make(map[string][]Diagnostic)
}

// Count returns the total number of diagnostics across all files.
func (dc *DiagnosticCache) Count() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	count := 0
	for _, v := range dc.diags {
		count += len(v)
	}
	return count
}

// Summary returns diagnostic counts grouped by severity level.
// Severities follow the LSP specification: 1=Error, 2=Warning, 3=Info, 4=Hint.
// Diagnostics without a severity are counted as errors.
func (dc *DiagnosticCache) Summary() (errors, warnings, infos, hints int) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	for _, diags := range dc.diags {
		for _, d := range diags {
			switch d.Severity {
			case DiagnosticSeverityWarning:
				warnings++
			case DiagnosticSeverityInfo:
				infos++
			case DiagnosticSeverityHint:
				hints++
			default:
				errors++
			}
		}
	}
	return
}
