package profile

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

// buildLayers writes optional profile.yaml files into the three layers and
// returns a loader over them. A non-empty profile writes a profile named
// "coder" in the profile root. The returned path is the coder profile dir.
func buildLayers(t *testing.T, profile, project, global string) (*Loader, string) {
	t.Helper()
	root := t.TempDir()
	profileRoot := filepath.Join(root, "profiles")
	projectDir := filepath.Join(root, "project")
	globalDir := filepath.Join(root, "global")
	write := func(dir, content string) {
		if content == "" {
			return
		}
		writeFile(t, filepath.Join(dir, "profile.yaml"), content)
	}
	write(projectDir, project)
	write(globalDir, global)
	if profile != "" {
		writeFile(t, filepath.Join(profileRoot, "coder", "profile.yaml"), profile)
	}
	return NewLoader(profileRoot, projectDir, globalDir), filepath.Join(profileRoot, "coder")
}

func TestResolveValidProfile(t *testing.T) {
	// REQ-PROFILE-1, valid profile: the resolved profile carries the declared
	// name, model, system prompt path, tools, skills, and permissions.
	loader, profileDir := buildLayers(t, `
name: coder
model: gpt-4o
system_prompt: SYSTEM.md
tools: [bash, read_file]
skills: [k8s]
permissions:
  - pattern: "*"
    action: deny
`, "", "")
	got, err := loader.Resolve("coder")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got.Name != "coder" || got.Model != "gpt-4o" {
		t.Errorf("resolved = %+v, want name coder and model gpt-4o", got)
	}
	if want := filepath.Join(profileDir, "SYSTEM.md"); got.SystemPrompt != want {
		t.Errorf("SystemPrompt = %q, want %q (resolved against the profile dir)", got.SystemPrompt, want)
	}
	if want := []string{"bash", "read_file"}; !reflect.DeepEqual(got.Tools, want) {
		t.Errorf("Tools = %v, want %v", got.Tools, want)
	}
	if want := []string{"k8s"}; !reflect.DeepEqual(got.Skills, want) {
		t.Errorf("Skills = %v, want %v", got.Skills, want)
	}
	if len(got.Permissions) != 1 || got.Permissions[0].Pattern != "*" || got.Permissions[0].Action != "deny" {
		t.Errorf("Permissions = %+v, want one deny-all rule", got.Permissions)
	}
}

func TestResolveMalformedYamlNamesFile(t *testing.T) {
	// REQ-PROFILE-1, malformed yaml: a typed error names the offending file.
	loader, _ := buildLayers(t, "name: [unclosed", "", "")
	_, err := loader.Resolve("coder")
	var actErr *core.ProfileActivationError
	if !errors.As(err, &actErr) {
		t.Fatalf("Resolve error = %v, want *core.ProfileActivationError", err)
	}
	if actErr.Name != "coder" {
		t.Errorf("ProfileActivationError.Name = %q, want %q", actErr.Name, "coder")
	}
	if !strings.HasSuffix(actErr.File, filepath.Join("profiles", "coder", "profile.yaml")) {
		t.Errorf("ProfileActivationError.File = %q, want it to name profile.yaml", actErr.File)
	}
}

func TestResolveUnknownProfile(t *testing.T) {
	// REQ-PROFILE-3, unknown profile: resolving a profile with no config
	// returns a typed unknown-profile error.
	loader, _ := buildLayers(t, "", "", "")
	_, err := loader.Resolve("nope")
	var unknown *core.UnknownProfileError
	if !errors.As(err, &unknown) {
		t.Fatalf("Resolve error = %v, want *core.UnknownProfileError", err)
	}
	if unknown.Name != "nope" {
		t.Errorf("UnknownProfileError.Name = %q, want %q", unknown.Name, "nope")
	}
}

func TestSystemPromptReadsBody(t *testing.T) {
	// REQ-PROFILE-1: the resolved SYSTEM.md body is returned verbatim.
	loader, profileDir := buildLayers(t, `
name: coder
system_prompt: SYSTEM.md
`, "", "")
	writeFile(t, filepath.Join(profileDir, "SYSTEM.md"), "You are coder.\n")
	got, err := loader.Resolve("coder")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	body, err := loader.SystemPrompt(got)
	if err != nil {
		t.Fatalf("SystemPrompt failed: %v", err)
	}
	if body != "You are coder.\n" {
		t.Errorf("body = %q, want %q", body, "You are coder.\n")
	}
}

func TestSystemPromptMissingBody(t *testing.T) {
	// REQ-PROFILE-1, missing SYSTEM.md: activation fails with a typed error
	// naming the missing file.
	loader, profileDir := buildLayers(t, `
name: coder
model: gpt-4o
system_prompt: SYSTEM.md
`, "", "")
	got, err := loader.Resolve("coder")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	_, err = loader.SystemPrompt(got)
	var actErr *core.ProfileActivationError
	if !errors.As(err, &actErr) {
		t.Fatalf("SystemPrompt error = %v, want *core.ProfileActivationError", err)
	}
	if want := filepath.Join(profileDir, "SYSTEM.md"); actErr.File != want {
		t.Errorf("ProfileActivationError.File = %q, want %q", actErr.File, want)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error = %v, want it to wrap os.ErrNotExist", err)
	}
}

func TestResolveOverridePrecedence(t *testing.T) {
	// REQ-PROFILE-2, override precedence: a profile declaring no model falls
	// back to the project layer, which wins over the global layer.
	loader, _ := buildLayers(t, `
name: coder
system_prompt: SYSTEM.md
`, "model: project-model\n", "model: global-model\n")
	got, err := loader.Resolve("coder")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got.Model != "project-model" {
		t.Errorf("Model = %q, want %q (project wins over global)", got.Model, "project-model")
	}
}

func TestResolveNearestWinsPerField(t *testing.T) {
	// REQ-PROFILE-2, nearest layer wins: the profile layer wins each field it
	// declares, including replacing lower-layer tool lists.
	loader, _ := buildLayers(t, `
name: coder
model: profile-model
tools: [bash]
`, "model: project-model\ntools: [read_file]\n", "model: global-model\n")
	got, err := loader.Resolve("coder")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got.Model != "profile-model" {
		t.Errorf("Model = %q, want %q (profile wins)", got.Model, "profile-model")
	}
	if want := []string{"bash"}; !reflect.DeepEqual(got.Tools, want) {
		t.Errorf("Tools = %v, want %v (profile tools replace lower layers)", got.Tools, want)
	}
}

func TestResolveEmptyLayersFallbackToGlobal(t *testing.T) {
	// REQ-PROFILE-2, empty profile layer: with no model in profile or project,
	// the global default is used.
	loader, _ := buildLayers(t, "name: coder\nsystem_prompt: SYSTEM.md\n", "", "model: global-default\n")
	got, err := loader.Resolve("coder")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got.Model != "global-default" {
		t.Errorf("Model = %q, want %q (global fallback)", got.Model, "global-default")
	}
}

func TestResolvePermissionsConcatLayers(t *testing.T) {
	// D15: permission rules from every layer concatenate global → project →
	// profile, so a profile rule with the same pattern overrides lower layers
	// at evaluation time (the ruleset's last-match-wins).
	loader, _ := buildLayers(t, `
name: coder
permissions:
  - pattern: bash
    action: deny
`, `permissions:
  - pattern: "*"
    action: allow
`, "")
	got, err := loader.Resolve("coder")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(got.Permissions) != 2 {
		t.Fatalf("Permissions length = %d, want 2 (project + profile)", len(got.Permissions))
	}
	if got.Permissions[0].Pattern != "*" || got.Permissions[0].Action != "allow" {
		t.Errorf("Permissions[0] = %+v, want the project allow-all rule first", got.Permissions[0])
	}
	if got.Permissions[1].Pattern != "bash" || got.Permissions[1].Action != "deny" {
		t.Errorf("Permissions[1] = %+v, want the profile bash deny rule last", got.Permissions[1])
	}
}

func TestDiscoverListsProfileNames(t *testing.T) {
	// REQ-PCLI-1: discover enumerates profile names whose profile.yaml exists
	// in the profile root, ignoring non-profile directories.
	loader, profileDir := buildLayers(t, "name: coder\n", "", "")
	writeFile(t, filepath.Join(filepath.Dir(profileDir), "writer", "profile.yaml"), "name: writer\n")
	writeFile(t, filepath.Join(filepath.Dir(profileDir), "empty", "nested.txt"), "not a profile")
	names, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if want := []string{"coder", "writer"}; !reflect.DeepEqual(names, want) {
		t.Errorf("Discover() = %v, want %v", names, want)
	}
}

func TestDiscoverEmptyRoot(t *testing.T) {
	// REQ-PCLI-1, no profiles: an empty (or absent) profile root discovers
	// nothing and is not an error.
	loader, _ := buildLayers(t, "", "", "")
	names, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("Discover() = %v, want empty", names)
	}
}
