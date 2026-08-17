# Proposal: Thinking Levels (--thinking flag)

## Intent

OpenAI reasoning models (o1, o3, o4-mini) support `reasoning_effort` to control how much "thinking" the model does before answering. kui has no way to expose this — the `StreamChunk.ReasoningDelta` field exists but is never populated, and there's no CLI or profile knob to set the level. Users running reasoning models waste tokens on high-effort thinking for simple tasks, or get shallow answers when they need deep reasoning.

## Scope

### In Scope
- `--thinking` CLI flag accepting level names: off, minimal, low, medium, high, xhigh, max
- `ThinkingLevelMap` — abstract levels mapped to provider-specific values (OpenAI: low/medium/high)
- `SetThinking(level string)` on `openai.Client` + `ReasoningEffort` field on `chatRequest`
- Per-profile default via `thinking` field in profile.yaml
- Per-run override via `--thinking` flag (CLI wins over profile)
- SSE parsing for `choices[].delta.reasoning_content` → `StreamChunk.ReasoningDelta`

### Out of Scope
- Anthropic extended thinking (different API shape)
- Google thinkingBudget (different API shape)
- TUI rendering of reasoning deltas (separate UI work)
- Provider-agnostic thinking interface (premature abstraction)

## Capabilities

### New Capabilities
None — this change is a thin extension of existing capabilities.

### Modified Capabilities
- `provider-openai-streaming`: Add `ReasoningEffort` to `chatRequest`, parse `reasoning_content` SSE deltas into `StreamChunk.ReasoningDelta`
- `cli-flags`: Add `Thinking string` field to `Options`, parse `--thinking` flag
- `profile-runtime`: Add `Thinking` field to `Config` and `Profile`, resolve in layered merge
- `profile-cli`: Add `kui profile thinking <name> <level>` subcommand

## Approach

Add `Thinking string` to `Options` and `Config`. Wire `--thinking` into `parseFlags`. In `runPrompt`, call `client.SetThinking(level)` after model resolution. The `chatRequest` struct gets an optional `ReasoningEffort string` field — only serialized when non-empty. SSE parser extracts `choices[].delta.reasoning_content` into `ReasoningDelta`. Level mapping: off/minimal/low → omit field; medium → "medium"; high → "high"; xhigh/max → "high" (OpenAI cap). Invalid levels fail with actionable error.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/kui/flags.go` | Modified | `Thinking string` in Options, `--thinking` in stringFlags |
| `internal/adapters/providers/openai/client.go` | Modified | `SetThinking()`, `ReasoningEffort` in chatRequest |
| `internal/adapters/providers/openai/sse.go` | Modified | Parse `reasoning_content` delta |
| `internal/adapters/profile/loader.go` | Modified | `Thinking` in Config/Profile, layered resolve |
| `cmd/kui/main.go` | Modified | Wire thinking level from opts to client |
| `cmd/kui/profile.go` | Modified | `kui profile thinking` subcommand |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| OpenAI changes reasoning_effort API | Low | Thin mapping layer; easy to update |
| Invalid thinking level passed to provider | Med | Validate in resolveThinking(); actionable error |
| Thinking ignored by non-reasoning models | Low | OpenAI silently ignores unknown fields |

## Rollback Plan

Remove `--thinking` flag, `SetThinking()`, `ReasoningEffort` field, and `Thinking` from Config/Profile. Restore original `chatRequest` struct. No data migration — thinking level is never persisted beyond profile.yaml.

## Dependencies

None. All changes are internal to kui.

## Success Criteria

- [ ] `kui --thinking medium "explain recursion"` sends `reasoning_effort: "medium"` in request body
- [ ] `kui --thinking off "hi"` omits `reasoning_effort` from request body
- [ ] Profile.yaml `thinking: high` is resolved and applied when no CLI flag is set
- [ ] CLI `--thinking` overrides profile default
- [ ] Invalid level (e.g., `--thinking banana`) returns actionable error
- [ ] SSE `reasoning_content` deltas populate `StreamChunk.ReasoningDelta`
- [ ] `go test ./...` passes
