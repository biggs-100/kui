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
