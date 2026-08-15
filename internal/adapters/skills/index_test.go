package skills

import (
	"errors"
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
