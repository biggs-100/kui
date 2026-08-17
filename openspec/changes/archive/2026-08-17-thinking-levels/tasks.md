# Tasks: Thinking Levels (--thinking flag)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 120-160 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | single PR |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

## Phase 1: CLI Flag & Validation

- [x] 1.1 RED: Add `Thinking string` to `Options` struct in `cmd/kui/flags.go`; add test in `flags_test.go` asserting `opts.Thinking == ""` on zero-value parse
- [x] 1.2 GREEN: Add `"thinking": true` to `stringFlags` map; add `setStringOption` case `"thinking"` → `opts.Thinking = value`
- [x] 1.3 RED: Add test `TestParseFlagsThinkingSpace` — `["--thinking", "high"]` → `opts.Thinking == "high"`
- [x] 1.4 GREEN: Already handled by stringFlags + setStringOption
- [x] 1.5 RED: Add test `TestParseFlagsThinkingEquals` — `["--thinking=medium"]` → `opts.Thinking == "medium"`
- [x] 1.6 RED: Add test `TestParseFlagsThinkingInvalid` — `["--thinking", "banana"]` returns error containing "banana" and listing valid values
- [x] 1.7 GREEN: Add `resolveThinking(level string) (string, error)` in `cmd/kui/main.go` — validates against {off, low, medium, high}; returns actionable error for invalid; returns "off" for empty
- [x] 1.8 REFACTOR: Update `usage` and `profileUsage` strings in `main.go` to document `--thinking` flag

## Phase 2: OpenAI Client Thinking

- [x] 2.1 RED: Add `thinkingLevel string` field to `Client` struct; add test asserting `SetThinking("high")` stores the value (export via getter or inspect via request body)
- [x] 2.2 GREEN: Add `func (c *Client) SetThinking(level string)` in `client.go` — same pattern as `SetModel`
- [x] 2.3 RED: Add `ReasoningEffort *string \`json:"reasoning_effort,omitempty"\`` to `chatRequest`; add test: when `thinkingLevel == "off"`, `ReasoningEffort` is nil; when `"high"`, pointer to `"high"`
- [x] 2.4 GREEN: Wire `thinkingLevel` → `ReasoningEffort` in `Chat()` and `StreamChat()` request builders
- [x] 2.5 RED: Add test in `client_test.go` (or new file): marshal `chatRequest` with nil `ReasoningEffort` → JSON has no `reasoning_effort` key; marshal with `&"medium"` → JSON has `"reasoning_effort":"medium"`

## Phase 3: SSE Reasoning Delta Parsing

- [x] 3.1 RED: Add `ReasoningContent string \`json:"reasoning_content"\`` to `streamDelta` in `sse.go`; add test: SSE chunk with `reasoning_content` → `StreamChunk.ReasoningDelta` populated
- [x] 3.2 GREEN: In `parseSSEChunk`, after text content check, add `if delta.ReasoningContent != "" { return core.StreamChunk{ReasoningDelta: delta.ReasoningContent}, true }`
- [x] 3.3 RED: Add test: SSE chunk without `reasoning_content` field → `ReasoningDelta` is empty string
- [x] 3.4 RED: Add test: SSE chunk with both `reasoning_content` and `content` → emit reasoning chunk first (current structure returns first match; adjust if needed)

## Phase 4: Profile Config & Layered Resolution

- [x] 4.1 RED: Add `Thinking string \`yaml:"thinking"\`` to `Config` struct; add `Thinking string` to `Profile` struct in `loader.go`; add test: `parseConfig` with `thinking: high` → `config.Thinking == "high"`
- [x] 4.2 GREEN: In `resolve()`, add `if p.Thinking == "" { p.Thinking = layer.config.Thinking }` alongside Model merge
- [x] 4.3 RED: Add test: profile.yaml with `thinking: medium` + project with `thinking: low` → resolved `Thinking == "medium"` (nearest wins)
- [x] 4.4 RED: Add test: no thinking in any layer → resolved `Thinking == ""` (default "off" at call site)

## Phase 5: CLI Wiring

- [x] 5.1 RED: In `runPrompt()`, add `client.SetThinking(resolveThinking(opts.Thinking, loader, activeName))` after model resolution; add test (httptest) that `--thinking medium` sends `reasoning_effort` in request body
- [x] 5.2 GREEN: Implement `resolveThinking` with layered resolution: if `opts.Thinking != ""` → use it; else if profile has thinking → use it; else → "off"
- [x] 5.3 RED: Add test: `--thinking` overrides profile thinking level
- [x] 5.4 RED: Add `kui profile thinking <name> <level>` subcommand in `runProfile` switch; add test: validates level, updates profile.yaml with `thinking: <level>`

## Phase 6: Verification

- [x] 6.1 Run `go test ./...` — all tests pass
- [x] 6.2 Run `go vet ./...` — no issues
- [x] 6.3 Manual: `kui --thinking medium "hi"` sends `reasoning_effort: "medium"` (inspect with --verbose or httptest)
- [x] 6.4 Manual: `kui --thinking off "hi"` omits `reasoning_effort` from request body
