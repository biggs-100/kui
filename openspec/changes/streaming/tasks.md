# Tasks: SSE Streaming for Real-Time Token-by-Token Responses

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 500–750 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Slice A) → PR 2 (Slice B) → PR 3 (Slice C) → PR 4 (Slice D) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| A | Core types + StreamingProvider + StreamingObserver interfaces | PR 1 | `go test ./internal/core/...` | N/A — pure type/interface definitions, no runtime | Remove `provider.go` additions and `observer.go` additions |
| B | OpenAI SSE StreamChat adapter | PR 2 | `go test ./internal/adapters/providers/openai/...` | `httptest.NewServer` with mock SSE responses | Remove `client.go` StreamChat + streaming types; revert to Chat-only |
| C | Agent loop streaming path | PR 3 | `go test ./internal/core/...` | Mock StreamingProvider returning controlled chunk sequences | Remove streaming path from `loop.go`; revert to sync-only |
| D | Controller + TUI wiring | PR 4 | `go test ./internal/tui/...` | Mock runner implementing/not implementing StreamingProvider | Remove streaming detection from `controller.go`; revert to sync runner |

## Phase 1: Core Types + Interfaces

- [x] 1.1 RED: `internal/core/stream_test.go` — test StreamChunk zero-value has no active fields; test StreamingProvider type assertion succeeds/fails
- [x] 1.2 GREEN: `internal/core/stream.go` — add StreamChunk struct (TextDelta, ReasoningDelta, ToolCallStart/Delta/End, Error, Done, Usage), add ToolCallDelta type, add Usage type, add IsTerminal() helper
- [x] 1.3 GREEN: `internal/core/streaming_provider_test.go` — implement tests: StreamingProvider interface satisfaction, non-streaming provider fails assertion, type assertion on concrete values
- [x] 1.4 Verify: `go test ./internal/core/...` — pass
- [x] 1.5 RED: `internal/core/streaming_observer_test.go` — test StreamingObserver type assertion; test emitTextDelta with nil observer (no panic); test emitTextDelta with non-nil observer (calls OnTextDelta)
- [x] 1.6 GREEN: `internal/core/streaming_provider.go` — add StreamingProvider interface with StreamChat method; `internal/core/streaming_observer.go` — add StreamingObserver interface with OnTextDelta(delta string); add emitTextDelta(observer, delta) nil-safe helper
- [x] 1.7 GREEN: `internal/core/stream_test.go` additional tests — IsTerminal() returns true for Done/Error, false for text/tool deltas
- [x] 1.8 Verify: `go test ./internal/core/... -race` — pass

## Phase 2: OpenAI SSE Adapter

- [ ] 2.1 RED: `internal/adapters/providers/openai/client_test.go` — test StreamChat with mock SSE server: normal text deltas parsed correctly; [DONE] sentinel closes channel; large event (200KB tool call) fits in 256KB buffer
- [ ] 2.2 GREEN: `internal/adapters/providers/openai/client.go` — add StreamChat method: POST with stream: true, create goroutine reading bufio.Scanner (256KB buffer), parse data: prefix lines, detect [DONE], unmarshal choices[].delta.content into StreamChunk, send to buffered chan(64), close channel on completion
- [ ] 2.3 GREEN: `internal/adapters/providers/openai/client.go` — add SSE response types (streamingDelta, streamingChoice, streamingResponse) for unmarshalling
- [ ] 2.4 GREEN: `internal/adapters/providers/openai/client.go` — add tool call accumulation: track index/name across chunks, emit ToolCallStart when function.name first seen, ToolCallDelta for arguments, ToolCallEnd on index change or stream end
- [ ] 2.5 Verify: `go test ./internal/adapters/providers/openai/...` — pass
- [ ] 2.6 RED: `internal/adapters/providers/openai/client_test.go` — test error mid-stream: SSE connection drops → StreamChunk{Error} sent, channel closed; test context cancellation stops stream; test no [DONE] before EOF sends error
- [ ] 2.7 GREEN: `internal/adapters/providers/openai/client.go` — handle mid-stream errors: send StreamChunk{Error: err}, close channel; handle context cancellation in scanner loop; handle EOF without [DONE] as error
- [ ] 2.8 Verify: `go test ./internal/adapters/providers/openai/...` — pass

## Phase 3: Agent Loop Streaming Path

- [ ] 3.1 RED: `internal/core/loop_test.go` — test streaming path: mock StreamingProvider returns [TextDelta("Hello"), TextDelta(" world"), Done]; verify observer.OnTextDelta called 2 times; verify final answer is "Hello world"
- [ ] 3.2 GREEN: `internal/core/loop.go` — in Run(), add type assertion `if sp, ok := provider.(StreamingProvider)`, call StreamChat, consume channel: forward TextDelta via emitTextDelta, accumulate tool calls, on Done close and execute accumulated tools
- [ ] 3.3 Verify: `go test ./internal/core/...` — pass
- [ ] 3.4 RED: `internal/core/loop_test.go` — test streaming with tool calls: mock returns [TextDelta, ToolCallStart, ToolCallDelta, ToolCallEnd, Done]; verify tool calls executed post-stream; verify loop continues with tool results
- [ ] 3.5 GREEN: `internal/core/loop.go` — implement tool call accumulation during streaming: track []ToolCall from ToolCallStart/Delta/End events, execute after channel closes, feed results back into loop
- [ ] 3.6 Verify: `go test ./internal/core/...` — pass
- [ ] 3.7 RED: `internal/core/loop_test.go` — test nil observer with streaming (no panic); test streaming error mid-stream (StreamChunk{Error} → loop returns error); test context cancellation closes stream
- [ ] 3.8 GREEN: `internal/core/loop.go` — ensure nil observer path is safe; handle StreamChunk{Error} by returning error; respect context cancellation
- [ ] 3.9 Verify: `go test ./internal/core/...` — pass

## Phase 4: Controller + TUI Wiring

- [ ] 4.1 RED: `internal/tui/controller_test.go` — test streaming path: mock runner implementing StreamingProvider, verify streamChunkMsg emitted per delta, verify streamDoneMsg on completion
- [ ] 4.2 GREEN: `internal/tui/controller.go` — in SubmitPrompt, add type assertion for StreamingProvider; if streaming: call StreamChat, consume channel in goroutine, emit streamChunkMsg for each TextDelta via tea.Cmd, emit streamDoneMsg on channel close
- [ ] 4.3 Verify: `go test ./internal/tui/...` — pass
- [ ] 4.4 RED: `internal/tui/controller_test.go` — test sync fallback: mock runner implementing only Provider (no StreamingProvider), verify Chat() called instead; test stream error → streamDoneMsg{err} emitted
- [ ] 4.5 GREEN: `internal/tui/controller.go` — ensure sync fallback path is unchanged; handle stream error: emit streamDoneMsg{err} on error chunk
- [ ] 4.6 Verify: `go test ./internal/tui/...` — pass
- [ ] 4.7 Verify ALL: `go test ./...` — full test suite pass
