package core

import "context"

// ProfileManager applies a queued profile switch between turns (D16,
// REQ-LOOP-5). ApplySwitch activates the named profile and returns the
// messages to append to the history before the next provider request: the
// new system prompt plus a profile-context marker identifying the newly
// active profile (REQ-LOOP-6). It never alters the conversation history.
type ProfileManager interface {
	ApplySwitch(ctx context.Context, name string) ([]Message, error)
}

// ModelMemory persists the per-profile model override (D17, REQ-CLI-4). The
// model resolution chain consults it before profile and global configuration.
type ModelMemory interface {
	Get(profile string) (string, bool)
	Set(profile, model string) error
}
