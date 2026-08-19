package lsp

import (
	"fmt"
	"sync"
	"testing"
)

func TestSetAndGet(t *testing.T) {
	cache := NewDiagnosticCache()

	diags := []Diagnostic{
		{
			Range:    Range{Start: Position{Line: 1, Character: 0}, End: Position{Line: 1, Character: 5}},
			Severity: DiagnosticSeverityError,
			Message:  "undefined: foo",
			Source:   "gopls",
		},
	}

	cache.Set("file:///tmp/test.go", diags)
	got := cache.Get("file:///tmp/test.go")

	if len(got) != 1 {
		t.Fatalf("Get() returned %d diagnostics, want 1", len(got))
	}
	if got[0].Message != "undefined: foo" {
		t.Errorf("message = %q, want %q", got[0].Message, "undefined: foo")
	}
}

func TestGetNonexistent(t *testing.T) {
	cache := NewDiagnosticCache()
	got := cache.Get("file:///nonexistent.go")
	if got != nil {
		t.Errorf("Get() on nonexistent key: got %v, want nil", got)
	}
}

func TestGetWorkspace(t *testing.T) {
	cache := NewDiagnosticCache()

	cache.Set("file:///a.go", []Diagnostic{{Message: "error in a"}})
	cache.Set("file:///b.go", []Diagnostic{{Message: "error in b"}, {Message: "warning in b"}})

	workspace := cache.GetWorkspace()
	if len(workspace) != 2 {
		t.Fatalf("GetWorkspace() returned %d files, want 2", len(workspace))
	}
	if len(workspace["file:///a.go"]) != 1 {
		t.Errorf("file a diagnostics: got %d, want 1", len(workspace["file:///a.go"]))
	}
	if len(workspace["file:///b.go"]) != 2 {
		t.Errorf("file b diagnostics: got %d, want 2", len(workspace["file:///b.go"]))
	}
}

func TestClear(t *testing.T) {
	cache := NewDiagnosticCache()
	cache.Set("file:///a.go", []Diagnostic{{Message: "error"}})
	cache.Set("file:///b.go", []Diagnostic{{Message: "error"}})

	cache.Clear("file:///a.go")

	if got := cache.Get("file:///a.go"); got != nil {
		t.Errorf("after Clear, Get(a) = %v, want nil", got)
	}
	if got := cache.Get("file:///b.go"); got == nil {
		t.Error("after Clear(a), Get(b) should still exist")
	}
}

func TestClearAll(t *testing.T) {
	cache := NewDiagnosticCache()
	cache.Set("file:///a.go", []Diagnostic{{Message: "error"}})
	cache.Set("file:///b.go", []Diagnostic{{Message: "error"}})

	cache.ClearAll()

	workspace := cache.GetWorkspace()
	if len(workspace) != 0 {
		t.Errorf("after ClearAll, workspace has %d files, want 0", len(workspace))
	}
}

func TestSetOverwrites(t *testing.T) {
	cache := NewDiagnosticCache()
	cache.Set("file:///a.go", []Diagnostic{{Message: "old"}})
	cache.Set("file:///a.go", []Diagnostic{{Message: "new"}, {Message: "new2"}})

	got := cache.Get("file:///a.go")
	if len(got) != 2 {
		t.Fatalf("after overwrite, got %d diagnostics, want 2", len(got))
	}
	if got[0].Message != "new" {
		t.Errorf("first message = %q, want %q", got[0].Message, "new")
	}
}

func TestCount(t *testing.T) {
	cache := NewDiagnosticCache()
	cache.Set("file:///a.go", []Diagnostic{{Message: "a1"}, {Message: "a2"}})
	cache.Set("file:///b.go", []Diagnostic{{Message: "b1"}})

	if got := cache.Count(); got != 3 {
		t.Errorf("Count() = %d, want 3", got)
	}
}

func TestSummary(t *testing.T) {
	cache := NewDiagnosticCache()
	cache.Set("file:///a.go", []Diagnostic{
		{Severity: DiagnosticSeverityError, Message: "error1"},
		{Severity: DiagnosticSeverityError, Message: "error2"},
		{Severity: DiagnosticSeverityWarning, Message: "warn1"},
		{Severity: DiagnosticSeverityInfo, Message: "info1"},
		{Severity: DiagnosticSeverityHint, Message: "hint1"},
	})
	cache.Set("file:///b.go", []Diagnostic{
		{Severity: DiagnosticSeverityWarning, Message: "warn2"},
		{Message: "no-severity"}, // defaults to error
	})

	errors, warnings, infos, hints := cache.Summary()
	if errors != 3 {
		t.Errorf("Summary() errors = %d, want 3", errors)
	}
	if warnings != 2 {
		t.Errorf("Summary() warnings = %d, want 2", warnings)
	}
	if infos != 1 {
		t.Errorf("Summary() infos = %d, want 1", infos)
	}
	if hints != 1 {
		t.Errorf("Summary() hints = %d, want 1", hints)
	}
}

func TestSummaryEmpty(t *testing.T) {
	cache := NewDiagnosticCache()
	errors, warnings, infos, hints := cache.Summary()
	if errors != 0 || warnings != 0 || infos != 0 || hints != 0 {
		t.Errorf("Summary() on empty cache = (%d,%d,%d,%d), want (0,0,0,0)", errors, warnings, infos, hints)
	}
}

func TestConcurrentAccess(t *testing.T) {
	cache := NewDiagnosticCache()
	const goroutines = 100
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Concurrent writers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				uri := fmt.Sprintf("file:///file%d.go", id)
				cache.Set(uri, []Diagnostic{{Message: fmt.Sprintf("diag-%d-%d", id, j)}})
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				uri := fmt.Sprintf("file:///file%d.go", id)
				cache.Get(uri)
				cache.GetWorkspace()
			}
		}(i)
	}

	// Concurrent clearers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				uri := fmt.Sprintf("file:///clear%d.go", id)
				cache.Set(uri, []Diagnostic{{Message: "temp"}})
				cache.Clear(uri)
			}
		}(i)
	}

	wg.Wait()
	// If we get here without -race detecting anything, the test passes
}
