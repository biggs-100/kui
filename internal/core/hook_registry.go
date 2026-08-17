package core

import "fmt"

// HookRegistry manages event-to-handler mappings, emitting hooks in
// registration order with error short-circuit semantics (D3, D7).
type HookRegistry struct {
	handlers map[string][]HookHandler
}

// NewHookRegistry returns an empty HookRegistry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		handlers: make(map[string][]HookHandler),
	}
}

// Register adds a handler for the given event. Registration is deterministic —
// handlers execute in the order they were registered. Registering a nil handler
// returns an error (REQ-HOOK-1).
func (r *HookRegistry) Register(event string, handler HookHandler) error {
	if r == nil {
		return fmt.Errorf("cannot register on nil HookRegistry")
	}
	if handler == nil {
		return fmt.Errorf("handler for event %q must not be nil", event)
	}
	r.handlers[event] = append(r.handlers[event], handler)
	return nil
}

// Emit calls all handlers registered for the event in registration order.
// When any handler returns a non-nil error, Emit stops executing remaining
// handlers and returns that error immediately (REQ-HOOK-2, REQ-HOOK-3).
// A nil *HookRegistry is a no-op that returns nil.
func (r *HookRegistry) Emit(event string, ctx HookContext) error {
	if r == nil {
		return nil
	}
	handlers := r.handlers[event]
	for _, handler := range handlers {
		if err := handler(ctx); err != nil {
			return fmt.Errorf("hook %q handler failed: %w", event, err)
		}
	}
	return nil
}

// HasHooks returns true if at least one handler is registered for the event
// (REQ-HOOK-4). A nil *HookRegistry returns false.
func (r *HookRegistry) HasHooks(event string) bool {
	if r == nil {
		return false
	}
	return len(r.handlers[event]) > 0
}
