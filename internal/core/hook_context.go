package core

// HookContext is the mutable context passed to hook handlers (D4, REQ-EXT-4).
// It provides access to messages, tool calls, and blocking capabilities.
type HookContext interface {
	// EventName returns the name of the event being processed.
	EventName() string

	// Messages returns the current message slice.
	Messages() []Message

	// SetMessages replaces the current message slice.
	SetMessages(messages []Message)

	// ToolCall returns the current tool call, if any.
	ToolCall() *ToolCall

	// SetToolCall replaces the current tool call.
	SetToolCall(call *ToolCall)

	// Block prevents downstream actions (e.g., tool execution) and stores
	// the reason.
	Block(reason string)

	// IsBlocked returns true if Block() was called.
	IsBlocked() bool

	// BlockReason returns the reason passed to Block(), or empty string.
	BlockReason() string
}

// hookContext is the concrete implementation of HookContext (D4).
type hookContext struct {
	eventName   string
	messages    []Message
	toolCall    *ToolCall
	blocked     bool
	blockReason string
}

// NewHookContext creates a new HookContext for the given event and messages.
func NewHookContext(event string, messages []Message) HookContext {
	return &hookContext{
		eventName: event,
		messages:  messages,
	}
}

func (c *hookContext) EventName() string { return c.eventName }

func (c *hookContext) Messages() []Message { return c.messages }

func (c *hookContext) SetMessages(messages []Message) {
	c.messages = messages
}

func (c *hookContext) ToolCall() *ToolCall { return c.toolCall }

func (c *hookContext) SetToolCall(call *ToolCall) {
	c.toolCall = call
}

func (c *hookContext) Block(reason string) {
	c.blocked = true
	c.blockReason = reason
}

func (c *hookContext) IsBlocked() bool { return c.blocked }

func (c *hookContext) BlockReason() string { return c.blockReason }
