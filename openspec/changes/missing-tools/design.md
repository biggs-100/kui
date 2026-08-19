# Design: Missing Tools — glob, grep, web_fetch

## Technical Approach

Three new tools added to `internal/adapters/tools/`, each implementing `core.Tool`. All file-based tools reuse the existing `resolvePath()` from `path.go` for workspace confinement (D11). Pure Go stdlib only — no external dependencies. Each tool follows the established pattern: JSON arg unmarshalling, path resolution, I/O, result formatting.

## Architecture Decisions

### Decision: Single-file per tool

**Choice**: One `.go` file per tool (glob.go, grep.go, web_fetch.go) with co-located `_test.go`
**Alternatives considered**: Single file for all three, or package per tool
**Rationale**: Matches existing pattern (file_sync.go, bash.go). Each tool is small enough (~80-120 lines) to live alone. Co-located tests follow the existing convention.

### Decision: filepath.WalkDir + filepath.Match for glob

**Choice**: `filepath.WalkDir` iterating entries, filter with `filepath.Match` per segment
**Alternatives considered**: `doublestar` library for `**` support, `fs.Glob`
**Rationale**: `filepath.Match` handles `*` and `?` but not `**` natively. We implement `**` by walking all directories and matching against the tail segments. Zero dependencies. `doublestar` is popular but adds a dependency for a simple feature.

### Decision: context timeout on web_fetch

**Choice**: `http.Client{Timeout: 30 * time.Second}` per request
**Alternatives considered**: Parent context deadline, configurable per-call timeout
**Rationale**: Spec mandates 30s hard timeout. Dedicated client avoids inheriting caller deadlines. Simple and explicit.

### Decision: binary file detection for grep

**Choice**: Check first 512 bytes for null bytes (standard heuristic)
**Alternatives considered**: `http.DetectContentType`, skip-by-extension
**Rationale**: Null-byte check is the standard approach (grep -I). Extension lists are brittle. Content-type detection adds imports for no benefit.

## Data Flow

### glob

```
Execute(pattern, path)
  → resolvePath(root, path)          // workspace check
  → filepath.WalkDir(resolved)
      → filepath.Match(pattern, rel)  // per-entry filter
  → sort.Strings(matches)
  → return JSON list
```

### grep

```
Execute(pattern, path, include, max_results)
  → resolvePath(root, path)           // workspace check
  → regexp.Compile(pattern)           // validate RE2
  → filepath.WalkDir(resolved)
      → filepath.Match(include, name) // optional file filter
      → isBinary(head) → skip
      → scanner line-by-line
      → regexp.MatchString(line)
  → truncate to max_results
  → return JSON list of "file:line:text"
```

### web_fetch

```
Execute(url, format)
  → url.Parse(url)                    // validate scheme
  → http.Client{Timeout: 30s}.Get(url)
  → io.ReadAll(resp.Body)
  → return body text (format=text)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/adapters/tools/glob.go` | Create | GlobTool — recursive file pattern matching |
| `internal/adapters/tools/glob_test.go` | Create | Tests: valid pattern, no matches, path escape, nested dirs |
| `internal/adapters/tools/grep.go` | Create | GrepTool — regex content search with include filter |
| `internal/adapters/tools/grep_test.go` | Create | Tests: regex match, no matches, binary skip, max results, path escape |
| `internal/adapters/tools/web_fetch.go` | Create | WebFetchTool — HTTP GET with timeout |
| `internal/adapters/tools/web_fetch_test.go` | Create | Tests: valid URL, timeout, invalid URL, network error |
| `internal/adapters/tools/registry.go` | Modify | Register glob, grep, web_fetch in built-in set |

## Interfaces / Contracts

Each tool follows the existing pattern. Constructor accepts workspace root (or http.Client for web_fetch).

```go
// glob.go
type GlobTool struct{ root string }
func NewGlob(root string) *GlobTool
// Schema: {"pattern": string (required), "path": string (optional)}

// grep.go
type GrepTool struct{ root string }
func NewGrep(root string) *GrepTool
// Schema: {"pattern": string (required), "path": string (optional),
//          "include": string (optional), "max_results": int (optional, default 100)}

// web_fetch.go
type WebFetchTool struct{ client *http.Client }
func NewWebFetch() *WebFetchTool
// Schema: {"url": string (required), "format": string (optional, "text"|"markdown")}
```

GrepTool result format per match: `"file:line:text"` (e.g. `"main.go:1:func main()"`).

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit (glob) | Pattern matching, nested dirs, path escape rejection, empty results | `t.TempDir()`, create fixture files, assert sorted output |
| Unit (grep) | Regex match, include filter, binary skip, max_results cap, path escape | `t.TempDir()`, fixture files with known content |
| Unit (web_fetch) | Valid URL, timeout, invalid scheme, network error | `httptest.NewServer` for success/error, `time.Sleep` for timeout test |
| Integration | All tools registered in registry, callable by name | Registry test already exists — add new tools to it |

Workspace confinement tests follow `TestReadFileEscapeRejected` pattern: write file outside root, assert `PathConstraintError`, assert no content leaked.

## Threat Matrix

N/A — no routing, shell commands, subprocesses, VCS/PR automation, executable-file classification, or process-integration boundary. File system access is confined to workspace root via `resolvePath()`.

## Migration / Rollout

No migration required. Pure additive change — three new tool files, one registry modification. Existing tools unaffected. `go build && go vet` confirms clean state.

## Open Questions

- [ ] Should `web_fetch` return raw HTML or strip tags for `format=text`? Spec says "page text content" — suggest stripping tags with `golang.org/x/net/html` or a simple regex. If no external deps allowed, return raw HTML and let the agent interpret.
- [ ] Should glob support `**` in the middle of a pattern (e.g. `src/**/*.go`) or only at the end? Proposal suggests full recursive support.
