package core

import (
	"context"
	"testing"
)

// mockStreamingProvider implements StreamingProvider for testing.
type mockStreamingProvider struct{}

func (m mockStreamingProvider) Chat(ctx context.Context, messages []Message, tools []Tool) ([]Message, error) {
	return nil, nil
}

func (m mockStreamingProvider) StreamChat(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	close(ch)
	return ch, nil
}

// mockNonStreamingProvider implements only Provider (no StreamChat).
type mockNonStreamingProvider struct{}

func (m mockNonStreamingProvider) Chat(ctx context.Context, messages []Message, tools []Tool) ([]Message, error) {
	return nil, nil
}

// TestStreamingProviderSatisfaction verifies that a type implementing both
// Provider.Chat and StreamingProvider.StreamChat satisfies StreamingProvider.
func TestStreamingProviderSatisfaction(t *testing.T) {
	var _ StreamingProvider = mockStreamingProvider{}
}

// TestNonStreamingProviderDoesNotSatisfyStreamingProvider verifies that a
// type implementing only Provider does not satisfy StreamingProvider.
func TestNonStreamingProviderDoesNotSatisfyStreamingProvider(t *testing.T) {
	var _ Provider = mockNonStreamingProvider{}

	var p Provider = mockNonStreamingProvider{}
	if _, ok := p.(StreamingProvider); ok {
		t.Error("non-streaming provider should not satisfy StreamingProvider via type assertion")
	}
}

// TestStreamingProviderTypeAssertion verifies type assertion works correctly
// on concrete and interface values.
func TestStreamingProviderTypeAssertion(t *testing.T) {
	// Concrete streaming provider
	sp := mockStreamingProvider{}
	if _, ok := Provider(sp).(StreamingProvider); !ok {
		t.Error("mockStreamingProvider should satisfy StreamingProvider")
	}

	// Concrete non-streaming provider
	nsp := mockNonStreamingProvider{}
	if _, ok := Provider(nsp).(StreamingProvider); ok {
		t.Error("mockNonStreamingProvider should not satisfy StreamingProvider")
	}
}

// TestStreamingProviderStreamChatReturnsChannel verifies StreamChat returns
// a non-nil channel and no error.
func TestStreamingProviderStreamChatReturnsChannel(t *testing.T) {
	sp := mockStreamingProvider{}
	ch, err := sp.StreamChat(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("StreamChat returned error: %v", err)
	}
	if ch == nil {
		t.Fatal("StreamChat returned nil channel")
	}
	// Drain the channel (it's closed in the mock)
	for range ch {
	}
}