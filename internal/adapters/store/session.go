// Package store implements session persistence for kui conversations.
// FileSessionStore writes sessions as JSON files under .kui/sessions/,
// with a metadata index for fast listing.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/biggs-100/kui/internal/core"
)

// sessionsDirName is the subdirectory under .kui that holds session files.
const sessionsDirName = "sessions"

// indexFileName is the metadata index for fast listing.
const indexFileName = "index.json"

// FileSessionStore implements core.SessionStore by persisting sessions as
// JSON files under {root}/.kui/sessions/. It writes atomically (temp + rename)
// and maintains a metadata index for fast listing without loading full messages.
type FileSessionStore struct {
	root string
}

// NewSessionStore creates a FileSessionStore rooted at the given directory.
// An empty root resolves from KUI_HOME, falling back to os.UserConfigDir()/kui.
func NewSessionStore(root string) *FileSessionStore {
	if root == "" {
		if home := os.Getenv("KUI_HOME"); home != "" {
			root = home
		} else if dir, err := os.UserConfigDir(); err == nil {
			root = filepath.Join(dir, "kui")
		} else {
			root = "."
		}
	}
	return &FileSessionStore{root: root}
}

// Rename sets a custom name on a session. It loads the session, updates the
// Name field in both the session file and the metadata index.
func (s *FileSessionStore) Rename(id string, name string) error {
	sess, err := s.Load(id)
	if err != nil {
		return err
	}

	sess.Meta.Name = name
	sess.Meta.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := s.Save(sess); err != nil {
		return err
	}
	return nil
}

// Save persists a session atomically: write to a temp file, then rename.
// It also updates the metadata index.
func (s *FileSessionStore) Save(session *core.Session) error {
	dir := s.sessionsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &core.StoreError{Op: "mkdir sessions", Err: err}
	}

	data, err := json.Marshal(session)
	if err != nil {
		return &core.StoreError{Op: "encode session", Err: err}
	}

	target := filepath.Join(dir, session.Meta.ID+".json")
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return &core.StoreError{Op: "write session tmp", Err: err}
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp) // best effort cleanup
		return &core.StoreError{Op: "rename session", Err: err}
	}

	if err := s.updateIndex(session.Meta); err != nil {
		return err
	}
	return nil
}

// Load reads a full session by ID from its JSON file.
func (s *FileSessionStore) Load(id string) (*core.Session, error) {
	path := filepath.Join(s.sessionsDir(), id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &core.StoreError{Op: "load session", Err: fmt.Errorf("session %q not found", id)}
		}
		return nil, &core.StoreError{Op: "read session", Err: err}
	}

	var sess core.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, &core.StoreError{Op: "parse session", Err: err}
	}
	return &sess, nil
}

// List returns metadata for all stored sessions, newest first.
// If the index is missing or corrupted, it rebuilds from session files on disk.
func (s *FileSessionStore) List() ([]core.SessionMeta, error) {
	metas, err := s.readIndex()
	if err != nil {
		// Index missing or corrupt — rebuild from session files.
		metas, err = s.rebuildIndex()
		if err != nil {
			return nil, err
		}
	}

	// Sort newest first by UpdatedAt (lexicographic works for RFC3339).
	// Falls back to CreatedAt when UpdatedAt is empty.
	sort.Slice(metas, func(i, j int) bool {
		iKey := metas[i].UpdatedAt
		if iKey == "" {
			iKey = metas[i].CreatedAt
		}
		jKey := metas[j].UpdatedAt
		if jKey == "" {
			jKey = metas[j].CreatedAt
		}
		return iKey > jKey
	})

	return metas, nil
}

// Delete removes a session file and its index entry.
func (s *FileSessionStore) Delete(id string) error {
	path := filepath.Join(s.sessionsDir(), id+".json")
	if _, err := os.Stat(path); err != nil {
		return &core.StoreError{Op: "delete session", Err: fmt.Errorf("session %q not found", id)}
	}

	if err := os.Remove(path); err != nil {
		return &core.StoreError{Op: "remove session file", Err: err}
	}

	if err := s.removeFromIndex(id); err != nil {
		return err
	}
	return nil
}

// GenerateSessionID produces a human-friendly session ID in the format
// profile-YYYY-MM-DD-HHMM-xxxx where xxxx is a random 4-char hex suffix
// to avoid collisions within the same minute.
func GenerateSessionID(profile string) string {
	now := time.Now().UTC()
	ts := now.Format("2006-01-02-1504")

	b := make([]byte, 2)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use timestamp-based pseudo-random (extremely unlikely path)
		b[0] = byte(now.Nanosecond() >> 8)
		b[1] = byte(now.Nanosecond())
	}
	suffix := hex.EncodeToString(b)

	return fmt.Sprintf("%s-%s-%s", profile, ts, suffix)
}

// --- internal helpers ---

func (s *FileSessionStore) sessionsDir() string {
	return filepath.Join(s.root, dirName, sessionsDirName)
}

func (s *FileSessionStore) indexPath() string {
	return filepath.Join(s.sessionsDir(), indexFileName)
}

// updateIndex merges metadata into the index file.
func (s *FileSessionStore) updateIndex(meta core.SessionMeta) error {
	metas, _ := s.readIndex() // ignore error — missing index is fine

	// Remove existing entry with same ID (upsert).
	updated := make([]core.SessionMeta, 0, len(metas)+1)
	for _, m := range metas {
		if m.ID != meta.ID {
			updated = append(updated, m)
		}
	}
	updated = append(updated, meta)

	return s.writeIndex(updated)
}

// removeFromIndex deletes a session entry from the index by ID.
func (s *FileSessionStore) removeFromIndex(id string) error {
	metas, _ := s.readIndex()

	updated := make([]core.SessionMeta, 0, len(metas))
	for _, m := range metas {
		if m.ID != id {
			updated = append(updated, m)
		}
	}

	return s.writeIndex(updated)
}

// readIndex loads the metadata index. Returns nil slice and error on failure.
func (s *FileSessionStore) readIndex() ([]core.SessionMeta, error) {
	data, err := os.ReadFile(s.indexPath())
	if err != nil {
		return nil, err
	}
	var metas []core.SessionMeta
	if err := json.Unmarshal(data, &metas); err != nil {
		return nil, err
	}
	return metas, nil
}

// writeIndex persists the full metadata index as JSON.
func (s *FileSessionStore) writeIndex(metas []core.SessionMeta) error {
	dir := s.sessionsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &core.StoreError{Op: "mkdir sessions for index", Err: err}
	}

	data, err := json.Marshal(metas)
	if err != nil {
		return &core.StoreError{Op: "encode index", Err: err}
	}

	if err := os.WriteFile(s.indexPath(), data, 0o644); err != nil {
		return &core.StoreError{Op: "write index", Err: err}
	}
	return nil
}

// rebuildIndex scans session files on disk and rebuilds the index.
func (s *FileSessionStore) rebuildIndex() ([]core.SessionMeta, error) {
	dir := s.sessionsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []core.SessionMeta{}, nil
		}
		return nil, &core.StoreError{Op: "read sessions dir for rebuild", Err: err}
	}

	var metas []core.SessionMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == indexFileName {
			continue
		}

		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // skip unreadable files
		}

		var sess core.Session
		if err := json.Unmarshal(data, &sess); err != nil {
			continue // skip corrupt files
		}
		metas = append(metas, sess.Meta)
	}

	// Persist the rebuilt index (best effort).
	_ = s.writeIndex(metas)

	return metas, nil
}
