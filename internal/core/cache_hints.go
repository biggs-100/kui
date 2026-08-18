package core

import "strings"

// CacheHint represents a cache control hint for a message.
type CacheHint struct {
	// Breakpoint marks this message as a cache breakpoint.
	// After this message, the provider should cache the prefix.
	Breakpoint bool
	// Key is an optional cache key for routing (OpenAI prompt_cache_key).
	Key string
}

// CacheAwareRequestBuilder optimizes message ordering for prompt caching.
// It ensures static content (system prompt, tools) comes first, followed
// by growing conversation history.
type CacheAwareRequestBuilder struct {
	systemPrompt string
	tools        []string // tool definitions as strings
	history      []Message
	currentTurn  []Message
}

// NewCacheAwareRequestBuilder creates a builder for cache-optimized requests.
func NewCacheAwareRequestBuilder() *CacheAwareRequestBuilder {
	return &CacheAwareRequestBuilder{}
}

// SetSystemPrompt sets the static system prompt (cacheable prefix).
func (b *CacheAwareRequestBuilder) SetSystemPrompt(prompt string) {
	b.systemPrompt = prompt
}

// SetTools sets the tool definitions (cacheable prefix).
func (b *CacheAwareRequestBuilder) SetTools(tools []string) {
	b.tools = tools
}

// SetHistory sets the conversation history (growing but prefix-stable).
func (b *CacheAwareRequestBuilder) SetHistory(history []Message) {
	b.history = history
}

// SetCurrentTurn sets the current turn messages.
func (b *CacheAwareRequestBuilder) SetCurrentTurn(messages []Message) {
	b.currentTurn = messages
}

// BuildCachePrefix returns the messages in cache-optimal order:
// 1. System prompt (static, cacheable)
// 2. Tools (static, cacheable)
// 3. History (growing, prefix-stable)
// 4. Current turn (changes every request)
func (b *CacheAwareRequestBuilder) BuildCachePrefix() []Message {
	var messages []Message

	// System prompt first (static).
	if b.systemPrompt != "" {
		messages = append(messages, Message{
			Role:    RoleSystem,
			Content: b.systemPrompt,
		})
	}

	// History (prefix-stable).
	messages = append(messages, b.history...)

	// Current turn last (changes every request).
	messages = append(messages, b.currentTurn...)

	return messages
}

// EstimateCacheHit estimates the cache hit ratio based on message stability.
// Returns 0.0-1.0 where 1.0 means perfect cache hit (all static content).
func (b *CacheAwareRequestBuilder) EstimateCacheHit() float64 {
	totalTokens := 0
	staticTokens := 0

	// System prompt is fully static.
	if b.systemPrompt != "" {
		tokens := estimateTokens(b.systemPrompt)
		totalTokens += tokens
		staticTokens += tokens
	}

	// Tools are static.
	for _, tool := range b.tools {
		tokens := estimateTokens(tool)
		totalTokens += tokens
		staticTokens += tokens
	}

	// History is prefix-stable (first N messages are static).
	for i, msg := range b.history {
		tokens := estimateTokens(msg.Content)
		totalTokens += tokens
		// Messages before the last 3 are considered static (prefix).
		if i < len(b.history)-3 {
			staticTokens += tokens
		}
	}

	// Current turn is not static.
	for _, msg := range b.currentTurn {
		totalTokens += estimateTokens(msg.Content)
	}

	if totalTokens == 0 {
		return 0
	}
	return float64(staticTokens) / float64(totalTokens)
}

// BuildCacheHint returns a CacheHint for the last static message.
// This is the optimal breakpoint for cache control.
func (b *CacheAwareRequestBuilder) BuildCacheHint() CacheHint {
	// The breakpoint should be after the last static content.
	// In practice, this is the system prompt or the last history message
	// before the current turn.
	return CacheHint{
		Breakpoint: true,
	}
}

// SortMessagesForCache reorders messages to maximize cache hits.
// Static content (system, tools) goes first, growing content goes last.
func SortMessagesForCache(messages []Message) []Message {
	if len(messages) == 0 {
		return messages
	}

	// Separate by role stability.
	var system, assistant, user, tool []Message
	for _, m := range messages {
		switch m.Role {
		case RoleSystem:
			system = append(system, m)
		case RoleAssistant:
			assistant = append(assistant, m)
		case RoleTool:
			tool = append(tool, m)
		case RoleUser:
			user = append(user, m)
		}
	}

	// Build cache-optimal order:
	// 1. System (static)
	// 2. Assistant history (prefix-stable)
	// 3. Tool results (prefix-stable)
	// 4. User messages (growing)
	var result []Message
	result = append(result, system...)
	result = append(result, assistant...)
	result = append(result, tool...)
	result = append(result, user...)

	return result
}

// IsCacheableRole reports whether a role is typically cacheable.
func IsCacheableRole(role string) bool {
	switch role {
	case RoleSystem, RoleAssistant:
		return true
	case RoleUser, RoleTool:
		return false // these change every request
	default:
		return false
	}
}

// CountCacheableTokens estimates how many tokens in a message sequence
// are likely to be cached by the provider.
func CountCacheableTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		if IsCacheableRole(m.Role) {
			total += estimateTokens(m.Content)
		}
	}
	return total
}

// EstimateCacheSavings estimates the cost savings from caching.
// Returns the fraction of input tokens that would be cached (0.0-1.0).
func EstimateCacheSavings(messages []Message) float64 {
	total := 0
	cacheable := 0
	for _, m := range messages {
		tokens := estimateTokens(m.Content)
		total += tokens
		if IsCacheableRole(m.Role) {
			cacheable += tokens
		}
	}
	if total == 0 {
		return 0
	}
	return float64(cacheable) / float64(total)
}

// BuildCacheKey generates a cache key from the session context.
// The key should be stable across requests with the same prefix.
func BuildCacheKey(profile, model, provider string) string {
	var sb strings.Builder
	if profile != "" {
		sb.WriteString(profile)
	}
	if model != "" {
		if sb.Len() > 0 {
			sb.WriteString(":")
		}
		sb.WriteString(model)
	}
	if provider != "" {
		if sb.Len() > 0 {
			sb.WriteString(":")
		}
		sb.WriteString(provider)
	}
	return sb.String()
}
