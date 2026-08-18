package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

// TestSaveCreatesFile covers task 2.1: saving a session creates a JSON file
// under .kui/sessions/{id}.json.
func TestSaveCreatesFile(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	sess := &core.Session{
		Meta: core.SessionMeta{
			ID:        "test-001",
			Profile:   "coder",
			CreatedAt: "2026-01-01T00:00:00Z",
		},
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "hello"},
		},
	}

	if err := ss.Save(sess); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	path := filepath.Join(root, ".kui", "sessions", "test-001.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("session file not created at %s: %v", path, err)
	}
}

// TestSaveUpdatesIndex covers task 2.3: saving a session creates/updates
// index.json with the session metadata.
func TestSaveUpdatesIndex(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	sess := &core.Session{
		Meta: core.SessionMeta{
			ID:        "test-002",
			Profile:   "coder",
			CreatedAt: "2026-01-01T00:00:00Z",
		},
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "hello"},
		},
	}

	if err := ss.Save(sess); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	indexPath := filepath.Join(root, ".kui", "sessions", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("index.json not created: %v", err)
	}

	if !strings.Contains(string(data), "test-002") {
		t.Errorf("index.json does not contain session ID test-002; content: %s", data)
	}
	if !strings.Contains(string(data), "coder") {
		t.Errorf("index.json does not contain profile coder; content: %s", data)
	}
}

// TestSaveCreatesAtomic covers task 2.2 triangulation: saving uses atomic
// write (temp file + rename), so no partial files exist after Save.
func TestSaveCreatesAtomic(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	sess := &core.Session{
		Meta: core.SessionMeta{
			ID:        "test-atomic",
			Profile:   "writer",
			CreatedAt: "2026-01-01T00:00:00Z",
		},
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "test atomic write"},
		},
	}

	if err := ss.Save(sess); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	sessionsDir := filepath.Join(root, ".kui", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Fatalf("cannot read sessions dir: %v", err)
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file found: %s", e.Name())
		}
	}
}

// TestLoadReturnsFullSession covers task 2.5: loading a saved session returns
// all messages and metadata.
func TestLoadReturnsFullSession(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	saved := &core.Session{
		Meta: core.SessionMeta{
			ID:        "test-003",
			Profile:   "coder",
			CreatedAt: "2026-06-15T10:30:00Z",
		},
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "what is go?"},
			{Role: core.RoleAssistant, Content: "Go is a programming language."},
			{Role: core.RoleUser, Content: "thanks"},
		},
	}

	if err := ss.Save(saved); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := ss.Load("test-003")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if loaded.Meta.ID != saved.Meta.ID {
		t.Errorf("Meta.ID = %q, want %q", loaded.Meta.ID, saved.Meta.ID)
	}
	if loaded.Meta.Profile != saved.Meta.Profile {
		t.Errorf("Meta.Profile = %q, want %q", loaded.Meta.Profile, saved.Meta.Profile)
	}
	if loaded.Meta.CreatedAt != saved.Meta.CreatedAt {
		t.Errorf("Meta.CreatedAt = %q, want %q", loaded.Meta.CreatedAt, saved.Meta.CreatedAt)
	}
	if len(loaded.Messages) != len(saved.Messages) {
		t.Fatalf("Messages len = %d, want %d", len(loaded.Messages), len(saved.Messages))
	}
	for i, msg := range loaded.Messages {
		if msg.Role != saved.Messages[i].Role {
			t.Errorf("Messages[%d].Role = %q, want %q", i, msg.Role, saved.Messages[i].Role)
		}
		if msg.Content != saved.Messages[i].Content {
			t.Errorf("Messages[%d].Content = %q, want %q", i, msg.Content, saved.Messages[i].Content)
		}
	}
}

// TestLoadNotFound covers triangulation for Load: loading a non-existent ID
// returns an error.
func TestLoadNotFound(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	_, err := ss.Load("does-not-exist")
	if err == nil {
		t.Fatal("Load for non-existent session should return error, got nil")
	}
}

// TestListReturnsMetadata covers task 2.7: List() returns SessionMeta entries
// for all saved sessions, sorted by created_at descending (newest first).
func TestListReturnsMetadata(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	sessions := []*core.Session{
		{
			Meta:     core.SessionMeta{ID: "s1", Profile: "coder", CreatedAt: "2026-01-01T00:00:00Z"},
			Messages: []core.Message{{Role: core.RoleUser, Content: "first"}},
		},
		{
			Meta:     core.SessionMeta{ID: "s2", Profile: "writer", CreatedAt: "2026-06-15T12:00:00Z"},
			Messages: []core.Message{{Role: core.RoleUser, Content: "second"}},
		},
		{
			Meta:     core.SessionMeta{ID: "s3", Profile: "coder", CreatedAt: "2026-03-10T08:00:00Z"},
			Messages: []core.Message{{Role: core.RoleUser, Content: "third"}},
		},
	}

	for _, s := range sessions {
		if err := ss.Save(s); err != nil {
			t.Fatalf("Save(%s) returned error: %v", s.Meta.ID, err)
		}
	}

	list, err := ss.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("List returned %d entries, want 3", len(list))
	}

	// Expect newest first: s2 (Jun), s3 (Mar), s1 (Jan)
	expectedOrder := []string{"s2", "s3", "s1"}
	for i, id := range expectedOrder {
		if list[i].ID != id {
			t.Errorf("List()[%d].ID = %q, want %q", i, list[i].ID, id)
		}
	}
}

// TestListEmpty covers triangulation for List: an empty store returns an
// empty slice, not nil.
func TestListEmpty(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	list, err := ss.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if list == nil {
		t.Fatal("List() returned nil, want empty slice")
	}
	if len(list) != 0 {
		t.Errorf("List() returned %d entries, want 0", len(list))
	}
}

// TestDeleteRemovesFileAndIndex covers task 2.9: deleting a session removes
// both the session file and its index entry.
func TestDeleteRemovesFileAndIndex(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	sess := &core.Session{
		Meta:     core.SessionMeta{ID: "del-001", Profile: "coder", CreatedAt: "2026-01-01T00:00:00Z"},
		Messages: []core.Message{{Role: core.RoleUser, Content: "to be deleted"}},
	}

	if err := ss.Save(sess); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := ss.Delete("del-001"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// File should be gone
	path := filepath.Join(root, ".kui", "sessions", "del-001.json")
	if _, err := os.Stat(path); err == nil {
		t.Error("session file still exists after Delete")
	}

	// Index entry should be gone
	list, err := ss.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	for _, m := range list {
		if m.ID == "del-001" {
			t.Error("deleted session still in index")
		}
	}
}

// TestDeleteNotFound covers triangulation for Delete: deleting a non-existent
// session returns an error.
func TestDeleteNotFound(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	err := ss.Delete("ghost")
	if err == nil {
		t.Fatal("Delete for non-existent session should return error, got nil")
	}
}

// TestIndexRebuiltOnDrift covers task 2.11: when the index file is missing,
// List() rebuilds it by scanning session files on disk.
func TestIndexRebuiltOnDrift(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	// Save two sessions to populate the index
	for _, id := range []string{"drift-a", "drift-b"} {
		sess := &core.Session{
			Meta:     core.SessionMeta{ID: id, Profile: "coder", CreatedAt: "2026-01-01T00:00:00Z"},
			Messages: []core.Message{{Role: core.RoleUser, Content: id}},
		}
		if err := ss.Save(sess); err != nil {
			t.Fatalf("Save(%s) returned error: %v", id, err)
		}
	}

	// Delete the index file to simulate drift
	indexPath := filepath.Join(root, ".kui", "sessions", "index.json")
	if err := os.Remove(indexPath); err != nil {
		t.Fatalf("failed to remove index file: %v", err)
	}

	// List should rebuild from session files
	list, err := ss.List()
	if err != nil {
		t.Fatalf("List after index drift returned error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List after drift returned %d entries, want 2", len(list))
	}

	// Index file should be recreated
	if _, err := os.Stat(indexPath); err != nil {
		t.Errorf("index.json not rebuilt after drift: %v", err)
	}

	// Verify both sessions are present
	ids := map[string]bool{}
	for _, m := range list {
		ids[m.ID] = true
	}
	if !ids["drift-a"] || !ids["drift-b"] {
		t.Errorf("missing sessions after rebuild: got IDs %v", ids)
	}
}

// TestIndexRebuiltOnCorrupt covers triangulation for index rebuild: a
// corrupted index file is also rebuilt from session files.
func TestIndexRebuiltOnCorrupt(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	sess := &core.Session{
		Meta:     core.SessionMeta{ID: "corrupt-1", Profile: "coder", CreatedAt: "2026-01-01T00:00:00Z"},
		Messages: []core.Message{{Role: core.RoleUser, Content: "data"}},
	}
	if err := ss.Save(sess); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Corrupt the index file
	indexPath := filepath.Join(root, ".kui", "sessions", "index.json")
	if err := os.WriteFile(indexPath, []byte("NOT VALID JSON"), 0o644); err != nil {
		t.Fatalf("failed to corrupt index: %v", err)
	}

	list, err := ss.List()
	if err != nil {
		t.Fatalf("List after corrupt index returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List after corrupt index returned %d entries, want 1", len(list))
	}
	if list[0].ID != "corrupt-1" {
		t.Errorf("List()[0].ID = %q, want %q", list[0].ID, "corrupt-1")
	}
}

// TestHumanFriendlyID covers task 2.13: GenerateSessionID produces the
// format profile-YYYY-MM-DD-HHMM with a 4-char hex suffix.
func TestHumanFriendlyID(t *testing.T) {
	id := GenerateSessionID("coder")

	// Format: coder-YYYY-MM-DD-HHMM-xxxx (hex suffix)
	parts := strings.Split(id, "-")
	if len(parts) < 6 {
		t.Fatalf("GenerateSessionID(%q) = %q, want format profile-YYYY-MM-DD-HHMM-xxxx", "coder", id)
	}

	if parts[0] != "coder" {
		t.Errorf("ID prefix = %q, want %q", parts[0], "coder")
	}

	// Last part should be 4-char hex
	suffix := parts[len(parts)-1]
	if len(suffix) != 4 {
		t.Errorf("hex suffix length = %d, want 4", len(suffix))
	}
	for _, c := range suffix {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("hex suffix contains non-hex char %q in %q", c, suffix)
			break
		}
	}
}

// TestHumanFriendlyIDUniqueness covers triangulation for GenerateSessionID:
// two calls in the same minute can collide on the timestamp, but the hex
// suffix makes them unique.
func TestHumanFriendlyIDUniqueness(t *testing.T) {
	ids := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := GenerateSessionID("coder")
		if ids[id] {
			t.Fatalf("duplicate ID generated: %q (iteration %d)", id, i)
		}
		ids[id] = true
	}
}

// TestSaveOverwritesExisting covers triangulation for Save: saving a session
// with the same ID overwrites the previous version cleanly.
func TestSaveOverwritesExisting(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	sess1 := &core.Session{
		Meta:     core.SessionMeta{ID: "overwrite-1", Profile: "coder", CreatedAt: "2026-01-01T00:00:00Z"},
		Messages: []core.Message{{Role: core.RoleUser, Content: "original"}},
	}
	if err := ss.Save(sess1); err != nil {
		t.Fatalf("Save (first) returned error: %v", err)
	}

	sess2 := &core.Session{
		Meta:     core.SessionMeta{ID: "overwrite-1", Profile: "coder", CreatedAt: "2026-01-01T00:00:00Z"},
		Messages: []core.Message{{Role: core.RoleUser, Content: "updated"}},
	}
	if err := ss.Save(sess2); err != nil {
		t.Fatalf("Save (second) returned error: %v", err)
	}

	loaded, err := ss.Load("overwrite-1")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Messages[0].Content != "updated" {
		t.Errorf("Messages[0].Content = %q, want %q", loaded.Messages[0].Content, "updated")
	}

	// Index should have exactly one entry for this ID
	list, err := ss.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	count := 0
	for _, m := range list {
		if m.ID == "overwrite-1" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("index has %d entries for overwrite-1, want 1", count)
	}
}

// TestMessageRoundTripThroughStore covers end-to-end: messages with ToolCall
// survive a full save→load cycle through the file store.
func TestMessageRoundTripThroughStore(t *testing.T) {
	root := t.TempDir()
	ss := NewSessionStore(root)

	saved := &core.Session{
		Meta:     core.SessionMeta{ID: "roundtrip-1", Profile: "coder", CreatedAt: "2026-01-01T00:00:00Z"},
		Messages: []core.Message{
			{Role: core.RoleUser, Content: "search for go generics"},
			{
				Role:    core.RoleAssistant,
				Content: "Let me search.",
				ToolCall: &core.ToolCall{
					ID:        "call_001",
					Name:      "web_search",
					Arguments: `{"query":"go generics"}`,
				},
			},
			{Role: core.RoleTool, Content: `{"results":["generics overview"]}`, ToolCallID: "call_001"},
			{Role: core.RoleAssistant, Content: "Generics in Go allow..."},
		},
	}

	if err := ss.Save(saved); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := ss.Load("roundtrip-1")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(loaded.Messages) != 4 {
		t.Fatalf("Messages len = %d, want 4", len(loaded.Messages))
	}

	// Check tool call survived round trip
	assistant := loaded.Messages[1]
	if assistant.ToolCall == nil {
		t.Fatal("ToolCall nil after round trip")
	}
	if assistant.ToolCall.ID != "call_001" {
		t.Errorf("ToolCall.ID = %q, want %q", assistant.ToolCall.ID, "call_001")
	}
	if assistant.ToolCall.Name != "web_search" {
		t.Errorf("ToolCall.Name = %q, want %q", assistant.ToolCall.Name, "web_search")
	}

	// Check tool result link
	toolResult := loaded.Messages[2]
	if toolResult.ToolCallID != "call_001" {
		t.Errorf("ToolCallID = %q, want %q", toolResult.ToolCallID, "call_001")
	}
}

// TestNewSessionStoreKUIHome covers the KUI_HOME override for session store:
// an empty root resolves to KUI_HOME env var.
func TestNewSessionStoreKUIHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("KUI_HOME", home)

	ss := NewSessionStore("")
	sess := &core.Session{
		Meta:     core.SessionMeta{ID: "kuihome-1", Profile: "coder", CreatedAt: "2026-01-01T00:00:00Z"},
		Messages: []core.Message{{Role: core.RoleUser, Content: "test"}},
	}
	if err := ss.Save(sess); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	path := filepath.Join(home, ".kui", "sessions", "kuihome-1.json")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("session file not created under KUI_HOME: %v", err)
	}
}
