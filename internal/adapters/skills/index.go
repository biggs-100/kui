// Package skills implements the on-demand skill index (D21, REQ-SKILL-1..3).
// Each skill lives in a per-skill directory under a layer's skills/ directory
// and holds a skill.yaml metadata file (name, description, triggers) plus a
// SKILL.md body. The index builds from the metadata alone — never reading any
// body (REQ-SKILL-2) — and reads a body only when the skill is invoked
// (REQ-SKILL-3). All filesystem and yaml work lives in this adapter.
package skills

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/biggs-100/kui/internal/core"
	"gopkg.in/yaml.v3"
)

// metadataFileName is the per-skill metadata file resolved in each skill
// directory (D21).
const metadataFileName = "skill.yaml"

// bodyFileName is the per-skill body file loaded only on invocation
// (D21, REQ-SKILL-3).
const bodyFileName = "SKILL.md"

// skillsDirName is the subdirectory under each layer root that holds the
// per-skill directories.
const skillsDirName = "skills"

// Skill is one indexed skill: the metadata from its skill.yaml plus the
// absolute path to its SKILL.md body (D21). The body text is never held in
// the index — Load reads it at invocation time (REQ-SKILL-3).
type Skill struct {
	Name        string
	Description string
	Triggers    []string
	BodyPath    string // absolute path to SKILL.md, loaded on invocation
	Layer       string // "global", "project" or "profile" — the source layer
}

// Match reports whether the skill applies to a message: true when at least
// one trigger appears in the message (case-insensitive) (REQ-SKILL-2).
func (s *Skill) Match(message string) bool {
	haystack := strings.ToLower(message)
	for _, trigger := range s.Triggers {
		if trigger != "" && strings.Contains(haystack, strings.ToLower(trigger)) {
			return true
		}
	}
	return false
}

// Meta is one layer's skill.yaml content (D21).
type Meta struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Triggers    []string `yaml:"triggers"`
}

// Index is the ordered, name-unique set of skills visible from the three
// layers (global → project → profile). When the same skill name appears in
// more than one layer the nearest layer wins (REQ-SKILL-1) and the shadowed
// layers are recorded in Collisions.
type Index struct {
	order      []string
	byName     map[string]*Skill
	Collisions []string
}

// NewIndex discovers skills across the global, remote, project and profile
// layer roots and builds an index WITHOUT reading any body file (REQ-SKILL-2).
// A skill named in more than one layer resolves to the nearest
// (profile > project > remote > global) and the shadowed layer is recorded in
// Collisions (REQ-SKILL-1, REQ-RS-15). Optional registryURLs are fetched
// concurrently; failures are logged and skipped (REQ-RS-18).
func NewIndex(globalDir, projectDir, profileDir string, registryURLs ...string) (*Index, error) {
	index := &Index{byName: make(map[string]*Skill)}

	// Layer 1: global
	if err := index.scanLayer(globalDir, "global"); err != nil {
		return nil, err
	}

	// Layer 2: remote (between global and project) — REQ-RS-15
	if len(registryURLs) > 0 {
		index.fetchRemoteRegistries(globalDir, registryURLs)
	}

	// Layer 3: project
	if err := index.scanLayer(projectDir, "project"); err != nil {
		return nil, err
	}

	// Layer 4: profile
	if err := index.scanLayer(profileDir, "profile"); err != nil {
		return nil, err
	}

	return index, nil
}

// fetchRemoteRegistries fetches skills from all registry URLs concurrently.
// Failures are logged as warnings and do not prevent local skills from loading
// (REQ-RS-4, REQ-RS-18).
func (idx *Index) fetchRemoteRegistries(globalDir string, registryURLs []string) {
	client := NewRegistryClient(10)
	cacheDir := filepath.Join(globalDir, "cache")
	cache := NewCache(cacheDir)

	for _, url := range registryURLs {
		idx.fetchRemote(context.Background(), client, cache, url)
	}
}

// fetchRemote fetches a skill index from one registry and downloads/caches
// each skill. Cached skills are reused when the version matches. All errors
// are logged and skipped (REQ-RS-4, REQ-RS-18).
func (idx *Index) fetchRemote(ctx context.Context, client *RegistryClient, cache *Cache, baseURL string) {
	index, err := client.FetchIndex(ctx, baseURL)
	if err != nil {
		log.Printf("kui: remote skills: fetch index from %s: %v", baseURL, err)
		return
	}

	hostname := extractHostname(baseURL)

	for _, skill := range index.Skills {
		prefixedName := hostname + "/" + skill.Name
		skillDir := cache.Dir(baseURL, skill.Name, skill.Version)

		if cache.IsCached(skillDir, skill.Version) {
			// Use cached version — scan from cache dir
			idx.scanCachedSkill(skillDir, prefixedName, skill)
			continue
		}

		// Download all files
		files := make(map[string][]byte)
		for _, file := range skill.Files {
			data, err := client.FetchFile(ctx, baseURL, skill.Name, file)
			if err != nil {
				log.Printf("kui: remote skills: fetch %s/%s: %v", skill.Name, file, err)
				continue
			}
			files[file] = data
		}

		if err := cache.Store(baseURL, skill.Name, skill.Version, files); err != nil {
			log.Printf("kui: remote skills: cache store %s: %v", skill.Name, err)
			continue
		}

		idx.scanCachedSkill(skillDir, prefixedName, skill)
	}
}

// scanCachedSkill indexes a single remote skill from its cached directory.
// It tries skill.yaml first, then falls back to frontmatter parsing of
// SKILL.md (REQ-RS-3, REQ-RS-20).
func (idx *Index) scanCachedSkill(skillDir, prefixedName string, entry IndexSkill) {
	// Try skill.yaml first
	yamlPath := filepath.Join(skillDir, metadataFileName)
	data, err := os.ReadFile(yamlPath)
	if err == nil {
		meta, err := parseMeta(data)
		if err == nil {
			bodyPath := filepath.Join(skillDir, bodyFileName)
			idx.add(&Skill{
				Name:        prefixedName,
				Description: meta.Description,
				Triggers:    meta.Triggers,
				BodyPath:    bodyPath,
				Layer:       "remote",
			}, "remote")
			return
		}
	}

	// Fall back to frontmatter of SKILL.md (REQ-RS-20)
	skillMDPath := filepath.Join(skillDir, bodyFileName)
	mdData, err := os.ReadFile(skillMDPath)
	if err != nil {
		// No skill.yaml and no SKILL.md — skip this skill
		return
	}
	fm, _, err := ParseFrontmatter(mdData)
	if err != nil {
		log.Printf("kui: remote skills: parse frontmatter %s: %v", skillMDPath, err)
		return
	}

	idx.add(&Skill{
		Name:        prefixedName,
		Description: fm.Description,
		Triggers:    fm.Triggers,
		BodyPath:    skillMDPath,
		Layer:       "remote",
	}, "remote")
}

// extractHostname extracts the hostname from a URL, stripping the scheme and
// port. For "https://registry.com:8080/skills" it returns "registry.com".
func extractHostname(url string) string {
	// Strip scheme
	host := url
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(host, prefix) {
			host = strings.TrimPrefix(host, prefix)
			break
		}
	}
	// Strip path
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	// Strip port
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}

// List returns the indexed skills in deterministic first-seen order.
func (i *Index) List() []*Skill {
	skills := make([]*Skill, 0, len(i.order))
	for _, name := range i.order {
		skills = append(skills, i.byName[name])
	}
	return skills
}

// Get resolves a skill by name.
func (i *Index) Get(name string) (*Skill, bool) {
	skill, ok := i.byName[name]
	return skill, ok
}

// Match returns the skills applicable to a message, in index order
// (REQ-SKILL-2).
func (i *Index) Match(message string) []*Skill {
	var matches []*Skill
	for _, name := range i.order {
		if skill := i.byName[name]; skill.Match(message) {
			matches = append(matches, skill)
		}
	}
	return matches
}

// Load reads a skill's SKILL.md body at invocation time (D21, REQ-SKILL-3).
// The index never contains body text; this is the only point the body is
// read. A missing or unreadable body returns a typed error naming the file.
func (i *Index) Load(skill *Skill) (string, error) {
	body, err := os.ReadFile(skill.BodyPath)
	if err != nil {
		return "", &core.SkillLoadError{Name: skill.Name, File: skill.BodyPath, Err: err}
	}
	return string(body), nil
}

// scanLayer indexes every skill directory under one layer root. A missing
// skills/ directory contributes nothing; a malformed skill.yaml fails the
// build with a typed error naming the file.
func (i *Index) scanLayer(dir, layer string) error {
	root := filepath.Join(dir, skillsDirName)
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		metaPath := filepath.Join(root, name, metadataFileName)
		data, err := os.ReadFile(metaPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue // a directory without skill.yaml is not a skill
			}
			return err
		}
		meta, err := parseMeta(data)
		if err != nil {
			return &core.SkillLoadError{Name: name, File: metaPath, Err: err}
		}
		i.add(&Skill{
			Name:        name,
			Description: meta.Description,
			Triggers:    meta.Triggers,
			BodyPath:    filepath.Join(root, name, bodyFileName),
			Layer:       layer,
		}, layer)
	}
	return nil
}

// add records a skill under its name. A nearer layer replaces a farther
// layer's entry and records the shadowing collision (REQ-SKILL-1); the
// first-seen position in order is preserved so ordering stays deterministic.
func (i *Index) add(skill *Skill, layer string) {
	prev, seen := i.byName[skill.Name]
	if seen {
		if prev.Layer == layer {
			return
		}
		i.Collisions = append(i.Collisions, fmt.Sprintf("skill %q from the %s layer shadows the %s layer entry", skill.Name, layer, prev.Layer))
	} else {
		i.order = append(i.order, skill.Name)
	}
	i.byName[skill.Name] = skill
}

// classifySkillsPaths separates a mixed list of skill entries into local
// directory names and remote registry URLs. HTTP/HTTPS entries are classified
// as remote; everything else is local (REQ-RS-13, REQ-RS-14).
func classifySkillsPaths(items []string) (dirs []string, urls []string) {
	for _, item := range items {
		if strings.HasPrefix(item, "http://") || strings.HasPrefix(item, "https://") {
			urls = append(urls, item)
		} else {
			dirs = append(dirs, item)
		}
	}
	return
}

// parseMeta parses one skill.yaml.
func parseMeta(data []byte) (*Meta, error) {
	var meta Meta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}
