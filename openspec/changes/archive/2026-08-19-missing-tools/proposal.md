# Proposal: Missing Tools — glob, grep, web_fetch

## Intent

The agent currently has read_file, write_file, and bash. For codebase exploration and web access, it must shell out to `find`, `grep`, or `curl` — slow, platform-fragile, and workspace-unaware. Adding native glob, grep, and web_fetch tools gives the agent fast, portable, workspace-confined search and fetch without external dependencies.

## Scope

### In Scope

- `glob` tool: recursive file pattern matching via `filepath.WalkDir` + `filepath.Match`
- `grep` tool: regex content search via `regexp` + `filepath.WalkDir`
- `web_fetch` tool: HTTP GET returning page text via `net/http`
- Tests for all three tools
- Registry update to expose new tools to the agent loop

### Out of Scope

- Shelling out to external `find`, `rg`, or `curl`
- Advanced regex features (backrefs, lookahead) — use Go `regexp` subset only
- HTML parsing, JavaScript rendering, or full DOM extraction
- Streaming or chunked responses for web_fetch

## Capabilities

### New Capabilities

None — all three tools extend the existing `agent-tools` capability.

### Modified Capabilities

- `agent-tools`: Three new requirements added (REQ-TOOLS-5 glob, REQ-TOOLS-6 grep, REQ-TOOLS-7 web_fetch). Existing requirements unchanged.

## Approach

Pure Go, zero external dependencies. Each tool follows the same pattern as existing tools:

- **glob**: `filepath.WalkDir` with `filepath.Match` on each path. Accept pattern string, return sorted file list. Constrain to workspace root.
- **grep**: `filepath.WalkDir` + `regexp.Compile` + line-by-line matching. Accept pattern and optional include glob. Return filename, line number, line text. Constrain to workspace root.
- **web_fetch**: `http.Get` with configurable timeout. Return response body as text. No workspace constraint (external resource).

All tools: synchronous, mandatory timeout, JSON parameter schema, workspace confinement where applicable.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/tools/` | New | Add `glob.go`, `grep.go`, `web_fetch.go` + `_test.go` each |
| `internal/tools/registry.go` | Modified | Register 3 new tools in the built-in set |
| `openspec/specs/agent-tools/spec.md` | Modified | Add REQ-TOOLS-5, REQ-TOOLS-6, REQ-TOOLS-7 |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Large directory trees cause slow glob/grep | Medium | Mandatory timeout per tool call; early termination on timeout |
| web_fetch to malicious URLs | Low | Timeout + response size cap; no JS execution |
| Path traversal in glob pattern | Low | Resolve against workspace root, reject escapes |

## Rollback Plan

Remove the three new tool files, revert registry changes, remove new requirements from agent-tools spec. Existing tools unaffected — pure additive change.

## Dependencies

None. Pure Go standard library only (`filepath`, `regexp`, `net/http`).

## Success Criteria

- [ ] `glob` returns matching files within workspace, rejects path escapes
- [ ] `grep` returns matching lines with file/line metadata, respects include pattern
- [ ] `web_fetch` returns HTTP response body, handles timeouts and errors
- [ ] All existing tests still pass
- [ ] `go build` clean, `go vet` clean
- [ ] Registry exposes all 6 tools (3 existing + 3 new)
