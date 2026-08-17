package core

import "context"

// StreamingProvider extends Provider with a streaming capability.
// Callers detect this interface via type assertion to opt into streaming.
// Non-streaming callers continue using Provider.Chat() unchanged (D1).
type StreamingProvider interface {
	Provider
	StreamChat(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error)
}