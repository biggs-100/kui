package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/biggs-100/kui/internal/core"
)

// TestSessionListCreate covers task 4.1: creating a SessionListModel
// from a list of SessionMeta produces a valid model with correct state.
func TestSessionListCreate(t *testing.T) {
	metas := []core.SessionMeta{
		{ID: "s1", Profile: "coder", Name: "Session One", CreatedAt: "2026-01-01T00:00:00Z"},
		{ID: "s2", Profile: "writer", Summary: "What is Go?", CreatedAt: "2026-06-15T12:00:00Z"},
	}

	model := NewSessionListModel(metas, 80, 24)

	if len(model.sessions) != 2 {
		t.Errorf("sessions len = %d, want 2", len(model.sessions))
	}
	if model.Selected() != "" {
		t.Errorf("Selected() = %q, want empty string on fresh model", model.Selected())
	}

	// View should render without panic
	view := model.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

// TestSessionListCreateEmpty covers edge case: empty session list.
func TestSessionListCreateEmpty(t *testing.T) {
	model := NewSessionListModel(nil, 80, 24)

	if len(model.sessions) != 0 {
		t.Errorf("sessions len = %d, want 0", len(model.sessions))
	}

	view := model.View()
	if view == "" {
		t.Error("View() returned empty string for empty list")
	}
}

// TestSessionListSelection covers task 4.3: pressing Enter selects the
// current session and returns its ID.
func TestSessionListSelection(t *testing.T) {
	metas := []core.SessionMeta{
		{ID: "s1", Profile: "coder", CreatedAt: "2026-01-01T00:00:00Z"},
		{ID: "s2", Profile: "writer", CreatedAt: "2026-06-15T12:00:00Z"},
	}

	model := NewSessionListModel(metas, 80, 24)

	// Press Enter — should select the first item (index 0)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if model.Selected() != "s1" {
		t.Errorf("Selected() = %q, want %q", model.Selected(), "s1")
	}
}

// TestSessionListNavigation covers arrow key navigation.
func TestSessionListNavigation(t *testing.T) {
	metas := []core.SessionMeta{
		{ID: "s1", Profile: "coder", CreatedAt: "2026-01-01T00:00:00Z"},
		{ID: "s2", Profile: "writer", CreatedAt: "2026-06-15T12:00:00Z"},
		{ID: "s3", Profile: "coder", CreatedAt: "2026-03-10T08:00:00Z"},
	}

	model := NewSessionListModel(metas, 80, 24)

	// Down arrow moves to next item
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	// Select the second item
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if model.Selected() != "s2" {
		t.Errorf("Selected() = %q, want %q after down+enter", model.Selected(), "s2")
	}
}

// TestSessionListEscape cancels selection.
func TestSessionListEscape(t *testing.T) {
	metas := []core.SessionMeta{
		{ID: "s1", Profile: "coder", CreatedAt: "2026-01-01T00:00:00Z"},
	}

	model := NewSessionListModel(metas, 80, 24)

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if model.Selected() != "" {
		t.Errorf("Selected() = %q, want empty after escape", model.Selected())
	}
}
