# Proposal: SSE Streaming for Real-Time Token-by-Token Responses

## Intent

kui currently waits for the full LLM response before rendering anything — users see a spinner, then the entire answer appears at once. Competing tools (pi, opencode, claude-code) stream token-by-token, making responses feel faster and more interactive. Streaming also enables progressive rendering of long responses and real-time status updates during generation.

## Scope

### In Scope
- `StreamingProvider` interface extending `Provider` (backward compatible, opt-in)
- `StreamChunk` type for streaming deltas (`Delta string`, `Done bool`, `Error error`)
- `OnTextDelta(delta string)` method on `Observer` interface
- OpenAI adapter `StreamChat()` implementation with SSE parsing (stdlib only)
- Controller detects streaming via type assertion, emits `streamChunkMsg` per delta
- Agent loop streaming path that processes `StreamChunk` channel

### Out of Scope
- Tool call streaming (Phase 2 — progressive argument display)
- Thinking/reasoning token streaming
- Multi-choice streaming support
- Non-OpenAI providers (future adapters)
- Backpressure tuning beyond drop-on-full (consistent with D3 pattern)

## Capabilities

### New Capabilities
- `streaming-text`: SSE streaming for text deltas, StreamingProvider interface, StreamChunk type
- `observer-text-delta`: OnTextDelta observer event for real-time text updates

### Modified Capabilities
- `provider-openai-compatible`: add StreamChat() with SSE parsing, Stream: true in request
- `agent-loop`: streaming-aware path that processes StreamChunk channel
- `tui-app`: handle streamChunkMsg per delta (existing type, already handled by AppendChunk)
- `tui-controller`: detect StreamingProvider, emit streamChunkMsg per delta

## Approach

**Interface Extension** — Add `StreamingProvider` interface extending `Provider`. Backward compatible, matches kui's existing `SetModeler` pattern for optional capability detection.

- Core types in `internal/core/provider.go`: `StreamChunk`, `StreamingProvider`
- Observer extension in `internal/core/observer.go`: `OnTextDelta(delta string)`
- Loop streaming path in `internal/core/loop.go`: type assertion, channel processing
- OpenAI adapter in `internal/adapters/providers/openai/client.go`: SSE parsing with `bufio.Scanner`
- Controller wiring in `internal/tui/controller.go`: detect streaming, emit per delta

**SSE Parsing**: Use `bufio.Scanner` with increased buffer (256KB) to handle large tool call JSON in events. Parse `data: ` prefix lines, detect `[DONE]` sentinel, unmarshal `choices[].delta.content`.

**Error Handling**: Stream errors arrive as `StreamChunk{Error: err}` and close the channel. Controller emits `streamDoneMsg{err}` on error.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/core/provider.go` | Modified | New `StreamChunk` type, new `StreamingProvider` interface |
| `internal/core/observer.go` | Modified | New `OnTextDelta(delta string)` method |
| `internal/core/loop.go` | Modified | Streaming-aware path with channel processing |
| `internal/adapters/providers/openai/client.go` | Modified | `StreamChat()` with SSE parsing |
| `internal/tui/controller.go` | Modified | Detect streaming, emit `streamChunkMsg` per delta |
| `internal/tui/app.go` | Modified | Handle streaming events (may need `streamTextDeltaMsg` type) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Scanner buffer overflow on large SSE events | Med | Increase buffer to 256KB; test with large tool call responses |
| Channel backpressure (TUI slow to consume) | Med | Drop-on-full pattern (consistent with D3); increase buffer if needed |
| Context cancellation during streaming | Low | Check `ctx.Err()` in SSE reader loop |
| Observer interface change is breaking | Low | Make it optional via separate interface or add default implementation |
| Error mid-stream (network failure) | Med | Send `StreamChunk{Error: err}`, close channel, emit `streamDoneMsg` |

## Rollback Plan

- Remove `StreamingProvider` interface and `StreamChunk` type from `core/provider.go`
- Revert `OnTextDelta` from Observer interface
- Revert loop to synchronous-only path
- Revert controller to synchronous runner
- Revert OpenAI adapter to `Chat()` only
- All changes are additive — existing synchronous `Provider.Chat()` unchanged
- Fallback: `git revert`

## Dependencies

- No new dependencies — stdlib only (`bufio`, `encoding/json`, `net/http`)
- Existing `StreamChunk` type in TUI already handles delta rendering via `AppendChunk()`

## Success Criteria

- [ ] `kui tui` streams text token-by-token (no spinner, progressive rendering)
- [ ] `Provider.Chat()` still works for non-streaming contexts (CLI mode, tests)
- [ ] OpenAI adapter handles SSE parsing correctly (including `[DONE]` sentinel)
- [ ] Observer receives `OnTextDelta()` calls during streaming
- [ ] Controller detects streaming capability and falls back gracefully
- [ ] `go test ./...` green; guard test blocks streaming deps in core

## Open Questions

1. **Observer breaking change**: Should `OnTextDelta()` be added to the main `Observer` interface (breaking change) or to a separate `StreamingObserver` interface (opt-in)? The exploration recommended main interface, but this breaks all existing Observer implementations.

2. **Scanner buffer size**: The exploration suggests 256KB, but SSE events with large tool calls could exceed this. Should we use `bufio.Reader.ReadLine()` instead for unbounded lines?

3. **Channel buffer size**: The exploration uses 32. Should we match the TUI's 64 (consistent with D3), or is 32 sufficient for streaming?

## Key Design Decisions

1. **Interface Extension** over replacing Provider interface — backward compatible, opt-in
2. **Phase 1 scope** — text streaming only, tool calls deferred to Phase 2
3. **OpenAI adapter first** — only provider implementing StreamingProvider initially
4. **Type assertion detection** — controller checks `if sp, ok := runner.(StreamingProvider)` like `SetModeler` pattern
5. **Drop-on-full** — consistent with TUI's existing channel pattern for backpressure
