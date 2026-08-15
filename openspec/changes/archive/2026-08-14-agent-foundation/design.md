# Design: Agent Foundation (kui)

## Technical Approach

Greenfield Go 1.26 module `github.com/biggs-100/kui`. Hexagonal: stdlib-only domain in `internal/core` (loop, ports, typed errors), adapters in `internal/adapters/`, wiring in `cmd/kui`. Strict TDD: each spec scenario → RED test first (`go test ./...`); verification per proposal (unit → httptest → smoke). ADRs at `docs/decisions/`.

## Architecture Decisions

| Decision | Option / tradeoff | Choice |
|---|---|---|
| D1 Boundary guard | CI grep vs test | `core/guard_test.go` runs `go list`; fails on non-stdlib core imports (ai-project-foundation recipe) |
| D2 Provider port | callback vs slice | `Chat(ctx, msgs, tools) ([]Message, error)` — full message slice |
| D3 Tool schema | struct vs string | `Schema() string` raw JSON; core stays schema-lib-free |
| D4 Registry | map vs ordered | ordered slice + lookup map; stable advertisement order |
| D5 Unknown tool | feed error back vs terminate | `UnknownToolError{Name}`, immediate return, no further requests (REQ-LOOP-2) |
| D6 Tool failure | raw vs wrapped | `ToolError{Name, Err}` (`%w`) — identifies failing tool (REQ-LOOP-3) |
| D7 Iteration budget | recover vs return | `IterationLimitError{Max}` after Max calls (REQ-LOOP-3) |
| D8 Env credentials | lazy vs construction-time | constructor fails naming `OPENAI_API_KEY`; key never in errors |
| D9 HTTP | custom vs stdlib | stdlib `net/http`, 60s timeout, Bearer auth (REQ-PROV-1) |
| D10 Error surface | one type vs typed | `AuthError`(401) / `RateLimitError`(429) / `ServerError`(5xx) / `TransportError` / `ParseError` (REQ-PROV-4) |
| D11 Path constraint | lexical vs +symlinks | Abs+EvalSymlinks+Rel check; reject before IO (REQ-TOOLS-1/2) |
| D12 bash | Run vs CommandContext | CommandContext + WithTimeout; Stdin nil; deadline → kill + TimeoutError (REQ-TOOLS-3) |
| D13 CLI exit codes | one vs distinct | 0 success / 1 runtime / 2 usage; answer stdout, errors stderr (REQ-CLI-1/2) |

## Data Flow

```
prompt → cmd/kui → core.Agent.Run ──▶ Provider.Chat ──▶ POST {base}/chat/completions
                                             ◀── messages (content | tool_calls)
  ◀── answer (no tool_calls → return)
       │ tool_calls → Registry.Get(name)
       │   known  → tool.Execute → tool-result Message → provider again
       │   unknown → UnknownToolError (terminate)
       │   budget → IterationLimitError (terminate)
```

## File Changes

All files are new (greenfield).

| File | Description |
|---|---|
| `go.mod`, `go.sum` | Module bootstrap, go 1.26, no runtime deps |
| `internal/core/errors.go` | UnknownToolError, IterationLimitError, ToolError |
| `internal/core/provider.go`, `tool.go` | Message/ToolCall types, Provider + Tool ports, Registry |
| `internal/core/loop.go` | Agent.Run loop with termination rules |
| `internal/core/loop_test.go`, `guard_test.go` | Fake provider/tool scenarios; stdlib-only guard |
| `internal/adapters/providers/openai/client.go` | Chat-completions client, env creds, typed errors |
| `internal/adapters/providers/openai/client_test.go` | httptest fixtures: tool-call seq, malformed JSON, 401/429/500, auth, base URL override |
| `internal/adapters/tools/path.go`, `read_file.go`, `write_file.go`, `bash.go`, `registry.go` | Built-in tools + default set |
| `internal/adapters/tools/*_test.go` | Temp-dir escapes, timeout kill, exit codes |
| `cmd/kui/main.go` | CLI wiring: provider + tools + loop |
| `docs/decisions/0001-hexagonal-layout.md`, `0002-env-based-credentials.md`, `0003-openai-compatible-only.md` | ADRs (ai-project-foundation template) |

## Interfaces / Contracts

```go
type Message struct {
    Role, Content string
    ToolCall     *ToolCall
    ToolCallID   string
}
type ToolCall struct{ ID, Name, Arguments string } // Arguments: raw JSON

type Provider interface {
    Chat(ctx context.Context, messages []Message, tools []Tool) ([]Message, error)
}
type Tool interface {
    Name() string; Description() string; Schema() string
    Execute(ctx context.Context, args json.RawMessage) (string, error)
}
```

`Agent.Run(ctx, prompt)`: append user msg; per iteration `Chat` → append assistant msgs; no tool_calls → return last content; else dispatch each (unknown → `UnknownToolError`; failure → `ToolError{Name}`), append tool results; bounded by `MaxIterations` → `IterationLimitError`.

## Testing Strategy

| Layer | What | How |
|---|---|---|
| Unit | Loop: direct, multi-step, unknown tool (typed, zero further calls), budget, failure identity; guard | Fake Provider/Tool in-memory |
| Unit | Tools: read/write in root, escape rejected, missing file, echo, non-zero exit, timeout kill | `t.TempDir()` roots; 1s timeout vs 10s sleep |
| Integration | Provider: tool-call seq, malformed JSON, 401/429/500, key present/absent, base URL | httptest fixtures asserting path/method/Bearer |
| Smoke | No-args usage (exit 2), missing key (exit 1) | `go run ./cmd/kui` subprocess |

## Threat Matrix

| Boundary | Applicability | Reason |
|---|---|---|
| Documentation-like paths | N/A | No doc-classification boundary |
| Git repository selection | N/A | No git automation in scope |
| Commit state | N/A | No commit automation |
| Push state | N/A | No push automation |
| PR commands | N/A | No PR automation |

bash creates the only subprocess boundary — covered by REQ-TOOLS-3 (timeout, no stdin, kill) with RED tests above.

## Migration / Rollout

No migration — greenfield. Rollback: drop branch / `git revert`. First apply task bootstraps `go.mod`.

## Open Questions

- [ ] `bash` on Windows dev machine (Git Bash in PATH) — smoke env note; CI documents dependency.
