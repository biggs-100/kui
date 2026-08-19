# Tasks: Missing Tools — glob, grep, web_fetch

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 500–630 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (glob) → PR 2 (grep) → PR 3 (web_fetch + registry) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Glob tool implementation | PR 1 | `go test ./internal/adapters/tools/... -run TestGlob` | N/A — pure file walk, no external I/O | `glob.go` + `glob_test.go` removable without affecting other tools |
| 2 | Grep tool implementation | PR 2 | `go test ./internal/adapters/tools/... -run TestGrep` | N/A — pure file walk + regexp | `grep.go` + `grep_test.go` removable |
| 3 | WebFetch tool + registry | PR 3 | `go test ./internal/adapters/tools/... -run TestWebFetch && go test ./internal/adapters/tools/... -run TestRegistry` | `httptest.NewServer` for success/error/timeout | `web_fetch.go` + `web_fetch_test.go` + registry changes removable |

---

## Phase 1: Glob Tool

- [x] 1.1 Create `internal/adapters/tools/glob.go` with `GlobTool` struct, `NewGlob(root string)` constructor, `Name()`, `Description()`, `Schema()` returning `{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string"}},"required":["pattern"]}`
- [x] 1.2 Implement `Execute`: unmarshal args, `resolvePath(root, path)`, `filepath.WalkDir` filtering with `filepath.Match` per segment, `sort.Strings(matches)`, return JSON list. Reject paths escaping workspace root (REQ-TOOLS-5)
- [x] 1.3 RED: Create `internal/adapters/tools/glob_test.go` — `TestGlobValidPattern` (two nested .go files, pattern `**/*.go`, assert both returned sorted)
- [x] 1.4 GREEN: Verify `TestGlobValidPattern` passes
- [x] 1.5 RED: `TestGlobNoMatches` — pattern `**/*.toml` on workspace with no .toml files, assert empty list
- [x] 1.6 RED: `TestGlobPathEscapeRejected` — path `../../secret`, assert `PathConstraintError`, assert no files read outside root
- [x] 1.7 GREEN: Verify tests from 1.5–1.6 pass

---

## Phase 2: Grep Tool

- [x] 2.1 Create `internal/adapters/tools/grep.go` with `GrepTool` struct, `NewGrep(root string)` constructor, `Name()`, `Description()`, `Schema()` returning `{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string"},"include":{"type":"string"},"max_results":{"type":"integer"}},"required":["pattern"]}`
- [x] 2.2 Implement `Execute`: unmarshal args, `resolvePath`, `regexp.Compile(pattern)` for RE2 validation, `filepath.WalkDir`, optional `filepath.Match(include, name)`, binary detection (null-byte in first 512 bytes), line-by-line scanner, truncate to `max_results` (default 100), return JSON list of `"file:line:text"` (REQ-TOOLS-6)
- [x] 2.3 RED: Create `internal/adapters/tools/grep_test.go` — `TestGrepValidRegex` (file with `func main()`, pattern `func\s+\w+`, include `*.go`, assert `main.go:1:func main()`)
- [x] 2.4 GREEN: Verify `TestGrepValidRegex` passes
- [x] 2.5 RED: `TestGrepNoMatches` — Go files with no `TODO`, pattern `TODO`, include `*.go`, assert empty list
- [x] 2.6 RED: `TestGrepBinarySkipped` — `image.png` with null bytes, pattern `.*`, assert binary file excluded
- [x] 2.7 RED: `TestGrepMaxResultsCap` — 200 matching lines across files, assert at most 100 returned
- [x] 2.8 RED: `TestGrepPathEscapeRejected` — path outside root, assert `PathConstraintError`
- [x] 2.9 GREEN: Verify tests from 2.5–2.8 pass

---

## Phase 3: WebFetch Tool

- [x] 3.1 Create `internal/adapters/tools/web_fetch.go` with `WebFetchTool` struct, `NewWebFetch()` constructor (no root needed), `Name()`, `Description()`, `Schema()` returning `{"type":"object","properties":{"url":{"type":"string"},"format":{"type":"string"}},"required":["url"]}`
- [x] 3.2 Implement `Execute`: unmarshal args, `url.Parse(url)` validate scheme (http/https only), `http.Client{Timeout: 30 * time.Second}.Get(url)`, `io.ReadAll(resp.Body)`, return body text (REQ-TOOLS-7)
- [x] 3.3 RED: Create `internal/adapters/tools/web_fetch_test.go` — `TestWebFetchValidURL` (httptest.NewServer serving HTML with `<title>`, assert body returned)
- [x] 3.4 GREEN: Verify `TestWebFetchValidURL` passes
- [x] 3.5 RED: `TestWebFetchInvalidURL` — `ftp://example.com`, assert URL validation error
- [x] 3.6 RED: `TestWebFetchNetworkError` — unreachable host, assert network error describing failure
- [x] 3.7 RED: `TestWebFetchTimeout` — server delays >30s, assert timeout error, no partial content
- [x] 3.8 GREEN: Verify tests from 3.5–3.7 pass

---

## Phase 4: Integration

- [x] 4.1 Update `internal/adapters/tools/registry.go` `Default`/`DefaultWithSyncer` to include `NewGlob(root)`, `NewGrep(root)`, `NewWebFetch()` in the returned slice
- [x] 4.2 RED: Update `internal/adapters/tools/registry_test.go` — add assertion that tool list contains glob, grep, web_fetch by name
- [x] 4.3 GREEN: Verify registry test passes
- [x] 4.4 Run full test suite: `go test ./internal/adapters/tools/...` — all tests pass
- [x] 4.5 Run `go vet ./internal/adapters/tools/...` — no warnings
