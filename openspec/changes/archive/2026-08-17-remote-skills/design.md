# Design: Remote Skill Fetching from Registries

## Technical Approach

Extend the existing 3-layer skills index (`global → project → profile`) with a 4th `remote` layer between global and project. A new `RegistryClient` port (hexagonal) fetches `index.json` manifests and per-skill files over HTTP. A disk cache with atomic swap avoids repeated downloads. The `skills: []string` profile config field accepts URLs alongside local names — no schema change. Local skills always work offline; registry failures are non-fatal warnings.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|-------------|-----------|
| Registry protocol | `GET {base}/index.json` + `GET {base}/{name}/{file}` | GraphQL, gRPC, custom | Matches OpenCode format; simple HTTP; stdlib-compatible |
| HTTP client | `net/http` stdlib | `resty`, `heimdall` | Project guards stdlib-only in core; no new deps |
| Cache key | SHA256 of `baseURL + skillName + version` | Content-hash, mtime | Version-aware; different versions produce different paths (REQ-RS-9) |
| Config field | Extend existing `skills: []string` | New `registries:` field | Zero schema change; URLs are just strings |
| Frontmatter | YAML frontmatter in `SKILL.md` | Separate `meta.json` | OpenCode-compatible; remote skills use same SKILL.md format |
| Collision logic | Same nearest-wins as local layers | Remote-only override | Consistent; profile/project always beat remote (REQ-SKILL-15) |
| Name prefixing | `{hostname}/{skillName}` for remote | No prefix | Prevents cross-registry collisions (REQ-SKILL-16) |

## Data Flow

```
Profile config (skills: [...])
        │
        ▼
classifySkillsPaths() → localNames []string, registryURLs []string
        │                          │
        ▼                          ▼
  scanLocal()              fetchRegistries()
  (global/project)         (concurrent HTTP)
        │                          │
        │                   ┌──────┴──────┐
        │                   ▼             ▼
        │            index.json    per-skill files
        │            (manifest)    (SKILL.md, skill.yaml)
        │                   │             │
        │                   ▼             ▼
        │              parse entries  atomic cache swap
        │              add to index   ({configRoot}/skills/cache/{sha256}/)
        │                   │             │
        ▼                   ▼             ▼
    ┌───────────────────────────────────────────┐
    │  Index.add() — 4 layers, nearest wins    │
    │  global → remote → project → profile     │
    └───────────────────────────────────────────┘
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/adapters/skills/registry.go` | Create | `RegistryClient` — HTTP fetch of index.json + skill files; context-aware |
| `internal/adapters/skills/registry_test.go` | Create | Tests with httptest server for protocol, errors, partial downloads |
| `internal/adapters/skills/cache.go` | Create | Disk cache: SHA256 paths, `.kui-version` tracking, atomic staging→rename |
| `internal/adapters/skills/cache_test.go` | Create | Cache hit/miss, atomic swap, corruption recovery |
| `internal/adapters/skills/frontmatter.go` | Create | Parse YAML frontmatter from SKILL.md for remote skills |
| `internal/adapters/skills/frontmatter_test.go` | Create | Frontmatter parsing edge cases |
| `internal/adapters/skills/index.go` | Modify | `NewIndex` accepts `[]RegistryURLs`; `scanRemote()` layer; `classifySkillsPaths()` |
| `internal/adapters/skills/index_test.go` | Modify | Add tests for 4-layer ordering, remote collision, URL classification |
| `internal/adapters/profile/loader.go` | Modify | No schema change needed — `Skills []string` already accepts any string |
| `cmd/kui/main.go` | Modify | Classify skills entries; pass registry URLs to `NewIndex` |

## Interfaces / Contracts

```go
// RegistryClient fetches skill manifests and files from a remote registry.
type RegistryClient struct {
    BaseURL    string
    HTTPClient *http.Client
}

// IndexEntry mirrors one entry in index.json (OpenCode-compatible).
type IndexEntry struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Triggers    []string `json:"triggers"`
    Version     string   `json:"version"`
    Files       []string `json:"files,omitempty"`
}

// RegistryIndex is the parsed index.json response.
type RegistryIndex struct {
    Skills []IndexEntry `json:"skills"`
}

// FetchIndex downloads and parses index.json from the registry.
func (rc *RegistryClient) FetchIndex(ctx context.Context) (*RegistryIndex, error)

// FetchFile downloads a single file from the registry.
func (rc *RegistryClient) FetchFile(ctx context.Context, skillName, filename string) ([]byte, error)

// Cache manages the disk cache for remote skills.
type Cache struct {
    Root string // {configRoot}/skills/cache/
}

// Dir returns the cache directory for a specific skill+version.
func (c *Cache) Dir(baseURL, skillName, version string) string

// IsCached checks if the cached version matches.
func (c *Cache) IsCached(baseURL, skillName, version string) bool

// Store atomically writes fetched files to cache.
func (c *Cache) Store(baseURL, skillName, version string, files map[string][]byte) error
```

## Modified NewIndex Signature

```go
// RemoteSkill represents one skill fetched from a registry.
type RemoteSkill struct {
    Name        string   // prefixed: "{hostname}/{skillName}"
    Description string
    Triggers    []string
    BodyPath    string   // cached SKILL.md path
    Layer       string   // "remote"
    RegistryURL string   // source registry base URL
}

// NewIndex discovers skills across global, remote, project, and profile layers.
// registryURLs are fetched concurrently; failures are logged and skipped.
func NewIndex(globalDir, projectDir, profileDir string, registryURLs []string) (*Index, error)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `RegistryClient` fetch/parse | `httptest.NewServer` serving fake index.json + files |
| Unit | `Cache` hit/miss/atomic swap | Temp dirs, verify `.kui-version`, verify no partial entries |
| Unit | `Frontmatter` parsing | Table-driven: valid, missing fields, no frontmatter |
| Unit | `classifySkillsPaths()` | Table-driven: URL vs name, edge cases |
| Integration | 4-layer index with remote | Mock registry + filesystem layers, verify collision order |
| Integration | Failure isolation | Registry timeout → warning log, local skills still present |
| E2E | `skills: ["https://..."]` | Full startup with registry, verify remote skills in List() |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. The `skills: []string` field already accepts any string — existing configs with only local names continue to work unchanged. Cache directory `{configRoot}/skills/cache/` is created on first fetch and is inert if removed.

## Open Questions

- [ ] Should `scanRemote()` use `errgroup` for concurrent registry fetches, or sequential with goroutines?
- [ ] Should the cache TTL be configurable, or stick with version-only invalidation (manual fetch on startup)?
- [ ] REQ-RS-16 says hostname prefix — should this be the full hostname or a shortened form (e.g., `example.com` not `https://example.com`)?
