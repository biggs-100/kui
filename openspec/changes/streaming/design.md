# Design: SSE Streaming for Real-Time Token-by-Token Responses

## Technical Approach

Add a `StreamingProvider` interface that extends the existing `Provider` port with `StreamChat()`, returning a channel of typed chunks. The OpenAI adapter implements it using stdlib `bufio.Scanner` for SSE parsing. The controller detects streaming capability via type assertion (matching the existing `SetModeler` pattern) and dispatches per-delta `streamChunkMsg` events to Bubble Tea. Tool calls accumulate during streaming and execute post-stream (Phase 1). Non-streaming providers remain fully backward compatible.

## Architecture Decisions

| # | Decision | Choice | Alternatives | Rationale |
|---|----------|--------|-------------|-----------|
| D1 | StreamingProvider interface | Extends Provider, opt-in via type assertion | Separate interface, new method on Provider | Backward compatible; matches existing `SetModeler` pattern; non-streaming callers unchanged |
| D2 | StreamChunk shape | Single struct with mutually-exclusive fields (TextDelta, ToolCallStart/Delta/DeltaEnd, Error, Done, Usage) | Separate chunk types per event | Simpler channel type; one-chunk-per-event convention; matches OpenAI SSE shape |
| D3 | Channel concurrency | Buffered chan(32), drop-on-full, context propagation | Unbuffered, blocking send, select loop | Consistent with Controller D3 pattern; bounded memory; no goroutine leaks |
| D4 | SSE parsing | `bufio.Scanner` with 256KB buffer, `data:` prefix strip | `bufio.Reader.ReadLine()`, custom parser | Stdlib-only; 256KB handles large tool call JSON; simpler than manual line protocol |
| D5 | Observer extension | Add `OnTextDelta(delta string)` to existing Observer interface | Separate StreamingObserver interface | Simpler; nil-safe via existing `emitObserver` recovery; one less interface to wire |
| D6 | Controller detection | Type assertion `if sp, ok := runner.(StreamingProvider)` | Feature flag, config flag | Zero-cost; matches `SetModeler` pattern; compile-time safety |
| D7 | Agent loop streaming path | Modify `Run()` to detect streaming provider internally, not a new `RunStream()` | Separate method | Single entry point; loop owns detection; controller doesn't need to know about streaming |
| D8 | Error propagation | `StreamChunk{Error: err}` then channel close | Error returned from StreamChat only | Allows mid-stream failures; consumer always gets terminal signal via channel close |

## Data Flow

```
Controller.SubmitPrompt()
  │
  ▼
Agent.Run()  ──→  detect StreamingProvider via type assertion
  │
  ▼ (streaming path)
Provider.StreamChat(ctx, messages, tools)
  │
  ▼
chan StreamChunk  ──→  Agent reads loop
  │                     │
  │  TextDelta ────────→ emitTextDelta(observer, delta)  ──→  OnTextDelta()
  │  ToolCallStart/Delta/End  →  accumulate in []ToolCall
  │  Error  ──────────→  send StreamChunk{Error}, close
  │  Done  ───────────→  close channel
  │
  ▼ (post-stream)
Execute accumulated tool calls  ──→  loop continues
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/core/provider.go` | Modify | Add `StreamChunk` struct, `ToolCallStart`/`ToolCallDelta`/`ToolCallEnd` types, `StreamingProvider` interface |
| `internal/core/observer.go` | Modify | Add `OnTextDelta(delta string)` method to Observer interface, add `emitTextDelta` helper |
| `internal/core/loop.go` | Modify | Streaming path in `Run()`: detect StreamingProvider, consume channel, forward deltas, accumulate tool calls |
| `internal/adapters/providers/openai/client.go` | Modify | Add `StreamChat()` with SSE parsing, streaming response types, `bufio.Scanner` with 256KB buffer |
| `internal/tui/controller.go` | Modify | `SubmitPrompt` detects streaming provider, consumes channel, emits per-delta `streamChunkMsg` |
| `internal/adapters/providers/openai/client_test.go` | Create | SSE parsing tests: normal events, [DONE] sentinel, large events, error mid-stream |

## Interfaces / Contracts

```go
// core/provider.go — new types
type StreamChunk struct {
    TextDelta    string
    ToolCallStart *ToolCallStart
    ToolCallDelta *ToolCallDelta
    ToolCallEnd   *ToolCallEnd
    Error         error
    Done          bool
    Usage         *Usage
}

type ToolCallStart struct {
    ID   string
    Name string
}

type ToolCallDelta struct {
    Index     int
    ID        string
    Name      string
    Arguments string
}

type ToolCallEnd struct {
    Index int
}

type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

type StreamingProvider interface {
    StreamChat(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error)
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | SSE line parsing, StreamChunk construction, channel lifecycle | Table-driven tests with mock SSE responses |
| Unit | Observer nil-safety for OnTextDelta | Call with nil observer, verify no panic |
| Integration | StreamingProvider type assertion in controller | Mock runner implementing/not implementing StreamingProvider |
| Integration | Agent loop streaming path | Mock StreamingProvider returning controlled chunks |
| E2E | OpenAI adapter StreamChat | Integration test with real API (skipped in CI) |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. All changes are additive:
- `StreamingProvider` is opt-in; existing `Provider.Chat()` callers unaffected
- Observer `OnTextDelta` is new; existing Observer implementations gain a method (compile-time catch)
- Controller falls back to synchronous path for non-streaming providers
- Feature can be toggled by swapping the adapter (non-streaming OpenAI client)

## Open Questions

- [ ] Should `OnTextDelta` be on the main Observer interface (breaking) or a separate `StreamingObserver` interface (opt-in)? Design assumes main interface per specs, but this breaks existing implementations. **Resolution needed before apply.**
