package core

import "fmt"

// UnknownToolError terminates the loop when the provider requests a tool that
// is not registered (D5, REQ-LOOP-2). No further provider requests are made.
type UnknownToolError struct {
	Name string
}

func (e *UnknownToolError) Error() string {
	return fmt.Sprintf("unknown tool %q", e.Name)
}

// IterationLimitError terminates the loop when the iteration budget is
// exhausted (D7, REQ-LOOP-3).
type IterationLimitError struct {
	Max int
}

func (e *IterationLimitError) Error() string {
	return fmt.Sprintf("iteration limit of %d reached", e.Max)
}

// ToolError wraps a tool execution failure and identifies the failing tool
// (D6, REQ-LOOP-3).
type ToolError struct {
	Name string
	Err  error
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("tool %q failed: %v", e.Name, e.Err)
}

// Unwrap exposes the underlying tool error to errors.Is / errors.As.
func (e *ToolError) Unwrap() error {
	return e.Err
}

// DuplicateToolError rejects registering two tools under the same name so the
// registry lookup map stays unambiguous (D4).
type DuplicateToolError struct {
	Name string
}

func (e *DuplicateToolError) Error() string {
	return fmt.Sprintf("tool %q already registered", e.Name)
}

// UnknownProfileError is returned when a profile switch targets a profile
// that is not registered (REQ-PROFILE-3).
type UnknownProfileError struct {
	Name string
}

func (e *UnknownProfileError) Error() string {
	return fmt.Sprintf("unknown profile %q", e.Name)
}

// PermissionError is returned when the loop is asked to dispatch a tool the
// active profile denies (D15, REQ-PERM-4).
type PermissionError struct {
	Tool string
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("tool %q not permitted", e.Tool)
}

// ProfileActivationError reports a profile that failed to activate, naming
// the profile and, when known, the file that caused the failure.
type ProfileActivationError struct {
	Name string
	File string
	Err  error
}

func (e *ProfileActivationError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("profile %q activation failed (%s): %v", e.Name, e.File, e.Err)
	}
	return fmt.Sprintf("profile %q activation failed: %v", e.Name, e.Err)
}

// Unwrap exposes the underlying activation failure.
func (e *ProfileActivationError) Unwrap() error {
	return e.Err
}

// SkillLoadError reports a skill that failed to load, naming the skill and
// the file that failed (REQ-SKILL-3).
type SkillLoadError struct {
	Name string
	File string
	Err  error
}

func (e *SkillLoadError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("skill %q failed to load (%s): %v", e.Name, e.File, e.Err)
	}
	return fmt.Sprintf("skill %q failed to load: %v", e.Name, e.Err)
}

// Unwrap exposes the underlying load failure.
func (e *SkillLoadError) Unwrap() error {
	return e.Err
}

// HookError wraps a hook handler failure with the event name so callers can
// identify which hook event triggered the error (REQ-HOOK-3).
type HookError struct {
	Event string
	Err   error
}

func (e *HookError) Error() string {
	return fmt.Sprintf("hook %q failed: %v", e.Event, e.Err)
}

// Unwrap exposes the underlying handler error to errors.Is / errors.As.
func (e *HookError) Unwrap() error {
	return e.Err
}

// StoreError wraps persistence failures of the profile store (REQ-PROFILE-4).
type StoreError struct {
	Op  string
	Err error
}

func (e *StoreError) Error() string {
	return fmt.Sprintf("store %s failed: %v", e.Op, e.Err)
}

// Unwrap exposes the underlying store failure.
func (e *StoreError) Unwrap() error {
	return e.Err
}
