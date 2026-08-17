package skills

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

// writeFile creates path (and its parents) with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// buildIndex creates the three layer roots under a temp dir and writes the
// given skills into them. It returns the three layer roots in order (global,
// project, profile). Each layer entry maps a skill name to a yaml string (and
// optionally a body content).
func buildIndex(t *testing.T, global, project, profile map[string]string) (groot, proot, froot string) {
	t.Helper()
	root := t.TempDir()
	groot = filepath.Join(root, "global")
	proot = filepath.Join(root, "project")
	froot = filepath.Join(root, "profile")
	writeLayer := func(dir string, skills map[string]string) {
		for name, yaml := range skills {
			writeFile(t, filepath.Join(dir, "skills", name, "skill.yaml"), yaml)
		}
	}
	writeLayer(groot, global)
	writeLayer(proot, project)
	writeLayer(froot, profile)
	return groot, proot, froot
}

// skillYAML builds a minimal skill.yaml body for a name, description and the
// given triggers.
func skillYAML(name, description string, triggers ...string) string {
	var b strings.Builder
	b.WriteString("name: " + name + "\n")
	b.WriteString("description: " + description + "\n")
	b.WriteString("triggers:\n")
	for _, t := range triggers {
		b.WriteString("  - " + t + "\n")
	}
	return b.String()
}

func TestIndexLayeredAggregation(t *testing.T) {
	// REQ-SKILL-1, layered aggregation: distinct skills in each of the three
	// layers are all present, each sourced from its own layer.
	groot, proot, froot := buildIndex(t,
		map[string]string{"go-testing": skillYAML("go-testing", "run go tests", "go test")},
		map[string]string{"k8s": skillYAML("k8s", "kubectl help", "kubectl")},
		map[string]string{"sql": skillYAML("sql", "sql help", "sql")},
	)
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	skills := index.List()
	if len(skills) != 3 {
		t.Fatalf("List() has %d skills, want 3 (one per layer)", len(skills))
	}
	byName := map[string]*Skill{}
	for _, s := range skills {
		byName[s.Name] = s
	}
	if got := byName["go-testing"].Layer; got != "global" {
		t.Errorf("go-testing Layer = %q, want %q", got, "global")
	}
	if got := byName["k8s"].Layer; got != "project" {
		t.Errorf("k8s Layer = %q, want %q", got, "project")
	}
	if got := byName["sql"].Layer; got != "profile" {
		t.Errorf("sql Layer = %q, want %q", got, "profile")
	}
}

func TestIndexCollisionNearestWins(t *testing.T) {
	// REQ-SKILL-1, collision resolution: when the same skill exists in the
	// global and profile layers, exactly one entry remains, sourced from the
	// nearest (profile) layer, and a collision diagnostic is recorded.
	groot, proot, froot := buildIndex(t,
		map[string]string{"go-testing": skillYAML("go-testing", "global copy", "go test")},
		map[string]string{},
		map[string]string{"go-testing": skillYAML("go-testing", "profile copy", "go test")},
	)
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	if len(index.List()) != 1 {
		t.Fatalf("List() has %d skills, want exactly 1 after collision", len(index.List()))
	}
	got, ok := index.Get("go-testing")
	if !ok {
		t.Fatal("Get(go-testing) not found")
	}
	if got.Layer != "profile" {
		t.Errorf("collision winner Layer = %q, want %q (nearest wins)", got.Layer, "profile")
	}
	if got.Description != "profile copy" {
		t.Errorf("winner Description = %q, want the profile copy", got.Description)
	}
	if len(index.Collisions) != 1 {
		t.Errorf("Collisions = %v, want one diagnostic", index.Collisions)
	}
}

func TestIndexCollisionProjectOverGlobal(t *testing.T) {
	// REQ-SKILL-1, three-layer collision: project shadows global but profile
	// still wins over both.
	groot, proot, froot := buildIndex(t,
		map[string]string{"tool": skillYAML("tool", "global", "tool")},
		map[string]string{"tool": skillYAML("tool", "project", "tool")},
		map[string]string{"tool": skillYAML("tool", "profile", "tool")},
	)
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	got, ok := index.Get("tool")
	if !ok || got.Layer != "profile" {
		t.Fatalf("Get(tool) = %+v, ok=%v, want the profile entry", got, ok)
	}
	if len(index.Collisions) != 2 {
		t.Errorf("Collisions = %v, want two diagnostics (global and project shadowed)", index.Collisions)
	}
}

func TestIndexBuildsWithoutBodies(t *testing.T) {
	// REQ-SKILL-2: the index builds from skill.yaml metadata alone; bodies are
	// never required (or read) while indexing.
	groot, proot, froot := buildIndex(t,
		map[string]string{"a": skillYAML("a", "skill a", "alpha")},
		map[string]string{},
		map[string]string{"b": skillYAML("b", "skill b", "beta")},
	)
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	if len(index.List()) != 2 {
		t.Fatalf("List() has %d skills, want 2 (no bodies present)", len(index.List()))
	}
}

func TestIndexTriggerMatch(t *testing.T) {
	// REQ-SKILL-2, trigger match: a message containing a trigger makes the
	// skill applicable.
	groot, proot, froot := buildIndex(t,
		map[string]string{"go-testing": skillYAML("go-testing", "run go tests", "go test")},
		map[string]string{},
		map[string]string{},
	)
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	matches := index.Match("please run go test on the loop package")
	if len(matches) != 1 || matches[0].Name != "go-testing" {
		t.Fatalf("Match() = %v, want the go-testing skill", matches)
	}
}

func TestIndexTriggerMatchCaseInsensitive(t *testing.T) {
	// REQ-SKILL-2, trigger matching is case-insensitive so natural-language
	// casing does not hide a skill.
	groot, proot, froot := buildIndex(t,
		map[string]string{"go-testing": skillYAML("go-testing", "run go tests", "Go Test")},
		map[string]string{},
		map[string]string{},
	)
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	if got := index.Match("run go test please"); len(got) != 1 {
		t.Errorf("Match(run go test please) = %v, want the go-testing skill (case-insensitive)", got)
	}
}

func TestIndexMatchNoTrigger(t *testing.T) {
	// REQ-SKILL-2, no trigger match: a message unrelated to every trigger
	// matches nothing, while a matching companion skill still matches.
	groot, proot, froot := buildIndex(t,
		map[string]string{"go-testing": skillYAML("go-testing", "run go tests", "go test")},
		map[string]string{},
		map[string]string{},
	)
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	if got := index.Match("what is the weather today?"); len(got) != 0 {
		t.Errorf("Match(unrelated) = %v, want no applicable skills", got)
	}
	if got := index.Match("run go test"); len(got) != 1 {
		t.Errorf("Match(go test) = %v, want the go-testing skill (control)", got)
	}
}

func TestIndexMatchEmptyIndex(t *testing.T) {
	// REQ-SKILL-2, empty index: with no skills, matching always returns none.
	groot, proot, froot := buildIndex(t, map[string]string{}, map[string]string{}, map[string]string{})
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	if got := index.Match("anything"); len(got) != 0 {
		t.Errorf("Match(anything) = %v, want no skills", got)
	}
}

func TestIndexLoadOnInvocation(t *testing.T) {
	// REQ-SKILL-3, body loads on invocation: the full SKILL.md content is read
	// only when requested.
	groot, proot, froot := buildIndex(t, map[string]string{}, map[string]string{}, map[string]string{})
	_ = groot
	dir := filepath.Join(proot, "skills", "go-testing")
	writeFile(t, filepath.Join(dir, "skill.yaml"), skillYAML("go-testing", "run go tests", "go test"))
	writeFile(t, filepath.Join(dir, "SKILL.md"), "# go-testing\n\nRun and debug Go tests.\n")
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	skill, ok := index.Get("go-testing")
	if !ok {
		t.Fatal("Get(go-testing) not found")
	}
	body, err := index.Load(skill)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if want := "# go-testing\n\nRun and debug Go tests.\n"; body != want {
		t.Errorf("Load() = %q, want the full SKILL.md body %q", body, want)
	}
}

func TestIndexLoadMissingBody(t *testing.T) {
	// REQ-SKILL-3, missing body: loading a skill whose SKILL.md does not exist
	// returns a typed error naming the missing file.
	groot, proot, froot := buildIndex(t, map[string]string{}, map[string]string{}, map[string]string{})
	dir := filepath.Join(groot, "skills", "ghost")
	writeFile(t, filepath.Join(dir, "skill.yaml"), skillYAML("ghost", "no body", "ghost"))
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	skill, ok := index.Get("ghost")
	if !ok {
		t.Fatal("Get(ghost) not found")
	}
	_, err = index.Load(skill)
	var loadErr *core.SkillLoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("Load error = %v, want *core.SkillLoadError", err)
	}
	if want := filepath.Join(groot, "skills", "ghost", "SKILL.md"); loadErr.File != want {
		t.Errorf("SkillLoadError.File = %q, want %q", loadErr.File, want)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Load error = %v, want it to wrap os.ErrNotExist", err)
	}
}

func TestIndexGetUnknown(t *testing.T) {
	// A lookup for an unindexed skill reports not-found.
	groot, proot, froot := buildIndex(t, map[string]string{}, map[string]string{}, map[string]string{})
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	if got, ok := index.Get("nope"); ok || got != nil {
		t.Errorf("Get(nope) = %v, ok=%v, want (nil, false)", got, ok)
	}
}

func TestIndexSkillMetadata(t *testing.T) {
	// REQ-SKILL-2: each skill exposes its name, description and triggers from
	// skill.yaml, with the body path pointing at the per-skill SKILL.md.
	groot, proot, froot := buildIndex(t,
		map[string]string{"go-testing": skillYAML("go-testing", "run go tests", "go test", "vet")},
		map[string]string{},
		map[string]string{},
	)
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	skill, ok := index.Get("go-testing")
	if !ok {
		t.Fatal("Get(go-testing) not found")
	}
	if skill.Name != "go-testing" || skill.Description != "run go tests" {
		t.Errorf("skill = {Name:%q Description:%q}, want name go-testing and description run go tests", skill.Name, skill.Description)
	}
	if want := []string{"go test", "vet"}; !reflect.DeepEqual(skill.Triggers, want) {
		t.Errorf("Triggers = %v, want %v", skill.Triggers, want)
	}
	if want := filepath.Join(groot, "skills", "go-testing", "SKILL.md"); skill.BodyPath != want {
		t.Errorf("BodyPath = %q, want %q", skill.BodyPath, want)
	}
}

// --- Phase 5: URL Classification (REQ-RS-13, REQ-RS-14) ---

func TestClassifySkillsPathsMixed(t *testing.T) {
	// REQ-RS-13, REQ-RS-14: mixed local names and registry URLs are separated.
	local, remote := classifySkillsPaths([]string{
		"go-testing",
		"https://example.com/skills/index.json",
		"k8s",
		"http://registry.local/index.json",
	})
	if len(local) != 2 || local[0] != "go-testing" || local[1] != "k8s" {
		t.Errorf("local = %v, want [go-testing k8s]", local)
	}
	if len(remote) != 2 || remote[0] != "https://example.com/skills/index.json" || remote[1] != "http://registry.local/index.json" {
		t.Errorf("remote = %v, want [https://example.com/skills/index.json http://registry.local/index.json]", remote)
	}
}

func TestClassifySkillsPathsEmpty(t *testing.T) {
	// REQ-RS-14: empty input returns two empty slices.
	local, remote := classifySkillsPaths(nil)
	if len(local) != 0 {
		t.Errorf("local = %v, want empty", local)
	}
	if len(remote) != 0 {
		t.Errorf("remote = %v, want empty", remote)
	}
}

func TestClassifySkillsPathsAllURLs(t *testing.T) {
	// REQ-RS-14: all-URLs input returns empty local, populated remote.
	local, remote := classifySkillsPaths([]string{
		"https://a.com/index.json",
		"https://b.com/index.json",
	})
	if len(local) != 0 {
		t.Errorf("local = %v, want empty", local)
	}
	if len(remote) != 2 {
		t.Errorf("remote has %d entries, want 2", len(remote))
	}
}

func TestClassifySkillsPathsAllNames(t *testing.T) {
	// REQ-RS-14: all-names input returns populated local, empty remote.
	local, remote := classifySkillsPaths([]string{"go-testing", "k8s"})
	if len(local) != 2 {
		t.Errorf("local has %d entries, want 2", len(local))
	}
	if len(remote) != 0 {
		t.Errorf("remote = %v, want empty", remote)
	}
}

// --- Phase 6: 4-Layer Index (REQ-RS-15) ---

func TestNewIndexNilRegistriesBackwardCompatible(t *testing.T) {
	// REQ-RS-17: NewIndex with nil registries works exactly as before.
	groot, proot, froot := buildIndex(t,
		map[string]string{"go-testing": skillYAML("go-testing", "run go tests", "go test")},
		map[string]string{},
		map[string]string{},
	)
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	skills := index.List()
	if len(skills) != 1 {
		t.Fatalf("List() has %d skills, want 1", len(skills))
	}
	if skills[0].Name != "go-testing" {
		t.Errorf("skill name = %q, want go-testing", skills[0].Name)
	}
}

func TestNewIndexEmptyRegistries(t *testing.T) {
	// REQ-RS-17: empty registry URLs list behaves like no registries.
	groot, proot, froot := buildIndex(t,
		map[string]string{"a": skillYAML("a", "skill a", "alpha")},
		map[string]string{},
		map[string]string{},
	)
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	if len(index.List()) != 1 {
		t.Fatalf("List() has %d skills, want 1", len(index.List()))
	}
}

func TestNewIndexWithRegistryURL(t *testing.T) {
	// REQ-RS-15: NewIndex with a registry URL adds a remote layer between
	// global and project. REQ-RS-16: remote skills get hostname prefix.
	groot, proot, froot := buildIndex(t,
		map[string]string{"global-skill": skillYAML("global-skill", "global", "global")},
		map[string]string{"project-skill": skillYAML("project-skill", "project", "project")},
		map[string]string{},
	)
	// Set up a mock registry serving one skill.
	srv := mockRegistryServer(t, RegistryIndex{
		Skills: []IndexSkill{
			{Name: "remote-skill", Version: "v1", Files: []string{"SKILL.md"}},
		},
	}, map[string]map[string][]byte{
		"remote-skill": {"SKILL.md": []byte("---\nname: remote-skill\ndescription: from remote\ntriggers:\n  - remote\n---\n# remote\n")},
	})
	defer srv.Close()

	index, err := NewIndex(groot, proot, froot, srv.URL)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	skills := index.List()
	if len(skills) != 3 {
		t.Fatalf("List() has %d skills, want 3 (global+remote+project): %v", len(skills), skills)
	}
	// Verify remote skill is present with correct layer (REQ-RS-16: hostname-prefixed).
	hostname := extractHostname(srv.URL)
	prefixed := hostname + "/remote-skill"
	remote, ok := index.Get(prefixed)
	if !ok {
		t.Fatalf("Get(%q) not found; available: %v", prefixed, index.List())
	}
	if remote.Layer != "remote" {
		t.Errorf("remote skill Layer = %q, want remote", remote.Layer)
	}
}

func TestNewIndexNilRegistriesParam(t *testing.T) {
	// REQ-RS-17: passing no registries works identically to before.
	groot, proot, froot := buildIndex(t,
		map[string]string{"x": skillYAML("x", "skill x", "x")},
		map[string]string{},
		map[string]string{},
	)
	index, err := NewIndex(groot, proot, froot)
	if err != nil {
		t.Fatalf("NewIndex(groot,proot,froot) returned error: %v", err)
	}
	if len(index.List()) != 1 {
		t.Errorf("List() has %d skills, want 1", len(index.List()))
	}
}

// --- Phase 6 continued: Collision resolution and failure isolation ---

func TestRemoteShadowsGlobal(t *testing.T) {
	// REQ-SKILL-1, REQ-RS-15: remote layer shadows global but is shadowed by
	// project and profile.
	groot, proot, froot := buildIndex(t,
		map[string]string{"tool": skillYAML("tool", "global", "tool")},
		map[string]string{},
		map[string]string{},
	)
	srv := mockRegistryServer(t, RegistryIndex{
		Skills: []IndexSkill{
			{Name: "tool", Version: "v1", Files: []string{"SKILL.md"}},
		},
	}, map[string]map[string][]byte{
		"tool": {"SKILL.md": []byte("---\ndescription: remote\ntoggles:\n  - tool\n---\n")},
	})
	defer srv.Close()

	index, err := NewIndex(groot, proot, froot, srv.URL)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	skill, ok := index.Get(extractHostname(srv.URL) + "/tool")
	if !ok {
		t.Fatal("remote tool skill not found")
	}
	if skill.Layer != "remote" {
		t.Errorf("Layer = %q, want remote (shadows global)", skill.Layer)
	}
	if skill.Description != "remote" {
		t.Errorf("Description = %q, want remote", skill.Description)
	}
}

func TestProjectShadowRemote(t *testing.T) {
	// REQ-SKILL-1, REQ-RS-15: project layer shadows remote layer.
	groot, proot, froot := buildIndex(t,
		map[string]string{},
		map[string]string{"tool": skillYAML("tool", "project", "tool")},
		map[string]string{},
	)
	srv := mockRegistryServer(t, RegistryIndex{
		Skills: []IndexSkill{
			{Name: "tool", Version: "v1", Files: []string{"SKILL.md"}},
		},
	}, map[string]map[string][]byte{
		"tool": {"SKILL.md": []byte("---\ndescription: remote\ntoggles:\n  - tool\n---\n")},
	})
	defer srv.Close()

	index, err := NewIndex(groot, proot, froot, srv.URL)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	skill, ok := index.Get("tool")
	if !ok {
		t.Fatal("Get(tool) not found")
	}
	if skill.Layer != "project" {
		t.Errorf("Layer = %q, want project (shadows remote)", skill.Layer)
	}
}

func TestProfileShadowRemote(t *testing.T) {
	// REQ-SKILL-1, REQ-RS-15: profile layer shadows remote layer.
	groot, proot, froot := buildIndex(t,
		map[string]string{},
		map[string]string{},
		map[string]string{"tool": skillYAML("tool", "profile", "tool")},
	)
	srv := mockRegistryServer(t, RegistryIndex{
		Skills: []IndexSkill{
			{Name: "tool", Version: "v1", Files: []string{"SKILL.md"}},
		},
	}, map[string]map[string][]byte{
		"tool": {"SKILL.md": []byte("---\ndescription: remote\ntoggles:\n  - tool\n---\n")},
	})
	defer srv.Close()

	index, err := NewIndex(groot, proot, froot, srv.URL)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	skill, ok := index.Get("tool")
	if !ok {
		t.Fatal("Get(tool) not found")
	}
	if skill.Layer != "profile" {
		t.Errorf("Layer = %q, want profile (shadows remote)", skill.Layer)
	}
}

func TestRegistryFailureLocalSkillsStillPresent(t *testing.T) {
	// REQ-RS-4, REQ-RS-18: when a registry URL fails, local skills are still
	// indexed and a warning is logged (not a fatal error).
	groot, proot, froot := buildIndex(t,
		map[string]string{"local-skill": skillYAML("local-skill", "local", "local")},
		map[string]string{},
		map[string]string{},
	)
	// Use an invalid URL that will fail to connect.
	index, err := NewIndex(groot, proot, froot, "http://127.0.0.1:1/nonexistent")
	if err != nil {
		t.Fatalf("NewIndex returned error: %v (registry failure should not be fatal)", err)
	}
	skills := index.List()
	if len(skills) != 1 {
		t.Fatalf("List() has %d skills, want 1 (local skill still present)", len(skills))
	}
	if skills[0].Name != "local-skill" {
		t.Errorf("skill name = %q, want local-skill", skills[0].Name)
	}
}

func TestRemoteSkillsAppearInList(t *testing.T) {
	// REQ-RS-19: remote skills appear in List() alongside local skills.
	groot, proot, froot := buildIndex(t,
		map[string]string{"local-a": skillYAML("local-a", "local a", "alpha")},
		map[string]string{"local-b": skillYAML("local-b", "local b", "beta")},
		map[string]string{},
	)
	srv := mockRegistryServer(t, RegistryIndex{
		Skills: []IndexSkill{
			{Name: "remote-x", Version: "v1", Files: []string{"SKILL.md"}},
		},
	}, map[string]map[string][]byte{
		"remote-x": {"SKILL.md": []byte("---\ndescription: remote x\ntriggers:\n  - gamma\n---\n")},
	})
	defer srv.Close()

	index, err := NewIndex(groot, proot, froot, srv.URL)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	skills := index.List()
	if len(skills) != 3 {
		t.Fatalf("List() has %d skills, want 3 (local-a + remote-x + local-b)", len(skills))
	}
	// Verify remote skill is in the list.
	names := make(map[string]bool)
	for _, s := range skills {
		names[s.Name] = true
	}
	hostname := extractHostname(srv.URL)
	if !names[hostname+"/remote-x"] {
		t.Errorf("remote-x not in List(): %v", skills)
	}
}

func TestRemoteSkillFrontmatterMetadata(t *testing.T) {
	// REQ-RS-20: frontmatter metadata is used when no skill.yaml for remote.
	groot, proot, froot := buildIndex(t, map[string]string{}, map[string]string{}, map[string]string{})
	srv := mockRegistryServer(t, RegistryIndex{
		Skills: []IndexSkill{
			{Name: "fm-skill", Version: "v1", Files: []string{"SKILL.md"}},
		},
	}, map[string]map[string][]byte{
		"fm-skill": {"SKILL.md": []byte("---\nname: fm-skill\ndescription: from frontmatter\ntriggers:\n  - frontmatter\n---\n# Body\n")},
	})
	defer srv.Close()

	index, err := NewIndex(groot, proot, froot, srv.URL)
	if err != nil {
		t.Fatalf("NewIndex returned error: %v", err)
	}
	hostname := extractHostname(srv.URL)
	skill, ok := index.Get(hostname + "/fm-skill")
	if !ok {
		t.Fatalf("Get(%s/fm-skill) not found", hostname)
	}
	if skill.Description != "from frontmatter" {
		t.Errorf("Description = %q, want 'from frontmatter'", skill.Description)
	}
	if len(skill.Triggers) != 1 || skill.Triggers[0] != "frontmatter" {
		t.Errorf("Triggers = %v, want [frontmatter]", skill.Triggers)
	}
	if skill.Layer != "remote" {
		t.Errorf("Layer = %q, want remote", skill.Layer)
	}
}

// mockRegistryServer starts an httptest server serving a fake registry with the
// given index and per-skill files.
func mockRegistryServer(t *testing.T, index RegistryIndex, skillFiles map[string]map[string][]byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(index)
	})
	for skillName, files := range skillFiles {
		for fileName, data := range files {
			p := "/" + skillName + "/" + fileName
			d := data // capture
			mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
				w.Write(d)
			})
		}
	}
	return httptest.NewServer(mux)
}
