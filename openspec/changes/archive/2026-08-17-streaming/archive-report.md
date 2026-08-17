# Archive Report: SSE Streaming for Real-Time Token-by-Token Responses

## Metadata

| Field | Value |
|-------|-------|
| Change | streaming |
| Archive Date | 2026-08-17 |
| Archived To | `openspec/changes/archive/2026-08-17-streaming/` |
| Artifact Store | openspec |
| Branch | main (merged via PR #28) |
| PRs | #24 (Slice A), #25 (Slice B), #26 (Slice C), #27 (Slice D), #28 (coverage gap unit E) |

## Verification

| Metric | Value |
|--------|-------|
| Verdict | PASS WITH WARNINGS |
| Requirements | 26/26 |
| Scenarios | 56/56 |
| Build | `go build ./...` — exit 0 |
| Vet | `go vet ./...` — exit 0 |
| Tests | `go test ./... -race -count=1` — exit 0, 11 packages OK |
| Evidence Revision | `sha256:dbf7799eae24adb642d99ea481acfc6e12bd7bd1dfbd5b7e74d981aba389fdd7` |
| Test Output Hash | `sha256:dbf7799eae24adb642d99ea481acfc6e12bd7bd1dfbd5b7e74d981aba389fdd7` |

### Coverage
| Package | Line % |
|---------|--------|
| internal/core | 94.2% |
| internal/adapters/providers/openai | 87.2% |
| internal/tui | 62.6% |
| internal/agent | 93.3% |
| internal/tui/views | 96.3% |

### Non-blocking Warnings (from verify-report)
1. StreamChunk type shape deviates from spec: `*core.ToolCall` reused for start/end instead of `*ToolCallStart`/`*ToolCallEnd`
2. Adapter-level accumulation is partial (ToolCallEnd not emitted by adapter; accumulation lives in loop)
3. Non-SSE JSON fallback lacks dedicated test
4. Design doc D5 records the abandoned choice (OnTextDelta on main Observer)
5. 4 lint findings in streaming-scope files (errcheck, ineffassign, unused field)
6. `internal/agent/agent.go` Provider() accessor at 0.0% coverage (trivial accessor)

## Tasks

**32/32 tasks complete** — all `[x]` in `tasks.md` (Phases 1–4, tasks 1.1–1.8, 2.1–2.8, 3.1–3.9, 4.1–4.7).

No unchecked implementation tasks. No stale-checkbox reconciliation required.

## Key Design Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | StreamingProvider extends Provider, opt-in via type assertion | Backward compatible; matches existing `SetModeler` pattern |
| D2 | StreamChunk single struct with mutually-exclusive fields | Simpler channel type; matches OpenAI SSE shape |
| D3 | Buffered chan(64), drop-on-full, context propagation | Consistent with Controller D3 pattern; bounded memory |
| D4 | `bufio.Scanner` with 256KB buffer for SSE parsing | Stdlib-only; handles large tool call JSON |
| D5 | Separate StreamingObserver interface (not main Observer) | Opt-in; nil-safe via existing emitObserver recovery |
| D6 | Controller type assertion for streaming detection | Zero-cost; compile-time safety |
| D7 | Loop detects streaming internally in Run() | Single entry point; loop owns detection |
| D8 | Error via StreamChunk{Error} then channel close | Allows mid-stream failures; consumer gets terminal signal |

## Specs Synced

5 new delta specs synced to main specs (all NEW, no existing main specs):

| Domain | Action | Details |
|--------|--------|---------|
| agent-loop-streaming | Created | 5 requirements, 12 scenarios |
| observer-streaming | Created | 4 requirements, 10 scenarios |
| provider-openai-streaming | Created | 6 requirements, 12 scenarios |
| provider-streaming | Created | 5 requirements, 10 scenarios |
| tui-streaming | Created | 6 requirements, 12 scenarios |

## Key Files Created/Modified

| File | Action | Description |
|------|--------|-------------|
| `internal/core/stream.go` | Created | StreamChunk struct, IsTerminal() helper |
| `internal/core/streaming_provider.go` | Created | StreamingProvider interface |
| `internal/core/streaming_observer.go` | Created | StreamingObserver interface, emitTextDelta |
| `internal/core/loop.go` | Modified | Streaming path: detect StreamingProvider, consume channel, accumulate tool calls |
| `internal/adapters/providers/openai/client.go` | Modified | StreamChat() with SSE parsing |
| `internal/adapters/providers/openai/sse.go` | Created | SSE line parsing, 256KB buffer |
| `internal/tui/controller.go` | Modified | Streaming detection, streamChunkMsg dispatch |
| `internal/tui/app.go` | Modified | Stream done/error handling |

## Source of Truth Updated

The following specs now reflect the new behavior:
- `openspec/specs/agent-loop-streaming/spec.md`
- `openspec/specs/observer-streaming/spec.md`
- `openspec/specs/provider-openai-streaming/spec.md`
- `openspec/specs/provider-streaming/spec.md`
- `openspec/specs/tui-streaming/spec.md`

## SDD Cycle Complete

The change has been fully planned, implemented, verified, and archived.
