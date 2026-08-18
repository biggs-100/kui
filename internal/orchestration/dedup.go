package orchestration

import (
	"crypto/sha256"
	"fmt"
	"sync"
)

// LaunchDedup prevents duplicate agent spawns and caches results.
type LaunchDedup struct {
	mu      sync.Mutex
	seen    map[string]bool
	results map[string]string // fingerprint → cached result
}

// NewLaunchDedup creates a new dedup instance.
func NewLaunchDedup() *LaunchDedup {
	return &LaunchDedup{
		seen:    make(map[string]bool),
		results: make(map[string]string),
	}
}

// IsDuplicate checks if this agent+task combination was already spawned.
// First call returns false and marks the combination as seen.
// Subsequent calls with the same combination return true.
func (d *LaunchDedup) IsDuplicate(agentName, task string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := d.fingerprint(agentName, task)
	return d.seen[key]
}

// MarkSeen marks a combination as seen and caches its result.
func (d *LaunchDedup) MarkSeen(agentName, task, result string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := d.fingerprint(agentName, task)
	d.seen[key] = true
	d.results[key] = result
}

// GetCached returns the cached result for a seen combination.
func (d *LaunchDedup) GetCached(agentName, task string) (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := d.fingerprint(agentName, task)
	if !d.seen[key] {
		return "", false
	}
	result, ok := d.results[key]
	return result, ok
}

// Reset clears the dedup cache (per session).
func (d *LaunchDedup) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen = make(map[string]bool)
	d.results = make(map[string]string)
}

// fingerprint creates a unique key for agent+task combination.
func (d *LaunchDedup) fingerprint(agentName, task string) string {
	h := sha256.Sum256([]byte(agentName + ":" + task))
	return fmt.Sprintf("%x", h)
}
