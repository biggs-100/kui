package views

import (
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/adapters/git"
)

func TestDiffModelCreate(t *testing.T) {
	m := NewDiffModel(testStyles())

	if len(m.Files()) != 0 {
		t.Errorf("expected empty file list, got %d", len(m.Files()))
	}
	if m.Cursor() != 0 {
		t.Errorf("expected zero cursor, got %d", m.Cursor())
	}
}

func TestDiffModelSetDiffs(t *testing.T) {
	m := NewDiffModel(testStyles())
	diffs := []git.FileDiff{
		{Path: "main.go", Status: "modified", Additions: 5, Deletions: 2},
		{Path: "helper.go", Status: "added", Additions: 10, Deletions: 0},
	}
	m.SetDiffs(diffs)

	if len(m.Files()) != 2 {
		t.Fatalf("expected 2 files, got %d", len(m.Files()))
	}
	if m.Files()[0].Path != "main.go" {
		t.Errorf("first file = %q, want %q", m.Files()[0].Path, "main.go")
	}
}

func TestDiffRender(t *testing.T) {
	m := NewDiffModel(testStyles())
	diffs := []git.FileDiff{
		{
			Path:      "main.go",
			Status:    "modified",
			Additions: 2,
			Deletions: 1,
			Hunks: []git.Hunk{
				{
					Header:   "@@ -10,7 +10,8 @@",
					OldStart: 10,
					NewStart: 10,
					Lines: []git.DiffLine{
						{Type: "context", Content: "package main"},
						{Type: "removed", Content: "fmt.Println(\"hello\")"},
						{Type: "added", Content: "fmt.Println(\"world\")"},
						{Type: "added", Content: "fmt.Println(\"done\")"},
					},
				},
			},
		},
	}
	m.SetDiffs(diffs)

	view := m.View()
	if !strings.Contains(view, "main.go") {
		t.Error("view should contain file name")
	}
	if !strings.Contains(view, "+2") {
		t.Error("view should contain addition count")
	}
	if !strings.Contains(view, "-1") {
		t.Error("view should contain deletion count")
	}
	if !strings.Contains(view, "@@") {
		t.Error("view should contain hunk header")
	}
	if !strings.Contains(view, "fmt.Println") {
		t.Error("view should contain diff content")
	}
}

func TestDiffViewEmpty(t *testing.T) {
	m := NewDiffModel(testStyles())
	view := m.View()
	if view == "" {
		t.Error("empty view should return non-empty string (hint)")
	}
}

func TestDiffNavigation(t *testing.T) {
	m := NewDiffModel(testStyles())
	diffs := []git.FileDiff{
		{Path: "a.go", Status: "modified", Additions: 1, Deletions: 0},
		{Path: "b.go", Status: "modified", Additions: 1, Deletions: 0},
		{Path: "c.go", Status: "modified", Additions: 1, Deletions: 0},
	}
	m.SetDiffs(diffs)

	// Move down
	m.MoveDown()
	if m.Cursor() != 1 {
		t.Errorf("cursor after MoveDown = %d, want 1", m.Cursor())
	}

	m.MoveDown()
	if m.Cursor() != 2 {
		t.Errorf("cursor after second MoveDown = %d, want 2", m.Cursor())
	}

	// Move down at end — should not go past
	m.MoveDown()
	if m.Cursor() != 2 {
		t.Errorf("cursor at end = %d, want 2", m.Cursor())
	}

	// Move up
	m.MoveUp()
	if m.Cursor() != 1 {
		t.Errorf("cursor after MoveUp = %d, want 1", m.Cursor())
	}
}

func TestDiffSelectedFile(t *testing.T) {
	m := NewDiffModel(testStyles())
	diffs := []git.FileDiff{
		{Path: "a.go", Status: "modified", Additions: 1, Deletions: 0},
		{Path: "b.go", Status: "added", Additions: 5, Deletions: 0},
	}
	m.SetDiffs(diffs)

	selected := m.SelectedFile()
	if selected == nil || selected.Path != "a.go" {
		t.Errorf("selected file = %v, want a.go", selected)
	}

	m.MoveDown()
	selected = m.SelectedFile()
	if selected == nil || selected.Path != "b.go" {
		t.Errorf("selected file = %v, want b.go", selected)
	}
}
