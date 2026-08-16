package agent

import (
	"testing"
)

// fakeModelMemory is a ModelMemory that returns pre-configured results.
type fakeModelMemory struct {
	models map[string]string
}

func (f *fakeModelMemory) Get(profile string) (string, bool) {
	m, ok := f.models[profile]
	return m, ok
}

// fakeModelLoader is a ModelLoader that returns pre-configured profiles.
type fakeModelLoader struct {
	profiles map[string]string // name → model
	errors   map[string]bool   // names that should error on Resolve
}

func (f *fakeModelLoader) Resolve(name string) (*ResolvedProfile, error) {
	if f.errors[name] {
		return nil, &modelError{name: name}
	}
	m, ok := f.profiles[name]
	if !ok {
		return &ResolvedProfile{}, nil
	}
	return &ResolvedProfile{Model: m}, nil
}

type modelError struct{ name string }

func (e *modelError) Error() string { return "unknown profile: " + e.name }

func TestResolveModelSavedModelWins(t *testing.T) {
	// REQ-CLI-4: saved model (ModelMemory) has highest priority.
	store := &fakeModelMemory{models: map[string]string{"coder": "claude-3-opus"}}
	loader := &fakeModelLoader{
		profiles: map[string]string{"coder": "gpt-4o"},
	}

	got := ResolveModel(store, loader, "coder", "env-model")
	if got != "claude-3-opus" {
		t.Errorf("ResolveModel = %q, want %q (saved should win)", got, "claude-3-opus")
	}
}

func TestResolveModelYamlModelSecondPriority(t *testing.T) {
	// REQ-CLI-4: yaml profile model is second priority when no saved model.
	store := &fakeModelMemory{models: map[string]string{}}
	loader := &fakeModelLoader{
		profiles: map[string]string{"coder": "gpt-4o"},
	}

	got := ResolveModel(store, loader, "coder", "env-model")
	if got != "gpt-4o" {
		t.Errorf("ResolveModel = %q, want %q (yaml model should be second priority)", got, "gpt-4o")
	}
}

func TestResolveModelEnvModelThirdPriority(t *testing.T) {
	// REQ-CLI-4: OPENAI_MODEL env var is third priority.
	store := &fakeModelMemory{models: map[string]string{}}
	loader := &fakeModelLoader{
		profiles: map[string]string{"coder": ""}, // yaml has no model
	}

	got := ResolveModel(store, loader, "coder", "env-model")
	if got != "env-model" {
		t.Errorf("ResolveModel = %q, want %q (env model should be third priority)", got, "env-model")
	}
}

func TestResolveModelDefaultFallback(t *testing.T) {
	// REQ-CLI-4: when nothing else is set, the built-in default is returned.
	store := &fakeModelMemory{models: map[string]string{}}
	loader := &fakeModelLoader{
		profiles: map[string]string{"coder": ""}, // yaml has no model
	}

	got := ResolveModel(store, loader, "coder", "")
	if got != "gpt-4o-mini" {
		t.Errorf("ResolveModel = %q, want %q (default fallback)", got, "gpt-4o-mini")
	}
}

func TestResolveModelUnknownProfileFallsThrough(t *testing.T) {
	// REQ-CLI-4: an unknown profile that errors on Resolve still falls
	// through to env/default.
	store := &fakeModelMemory{models: map[string]string{}}
	loader := &fakeModelLoader{
		errors: map[string]bool{"nope": true},
	}

	got := ResolveModel(store, loader, "nope", "env-model")
	if got != "env-model" {
		t.Errorf("ResolveModel = %q, want %q (unknown profile falls through to env)", got, "env-model")
	}
}

func TestResolveModelSavedModelIgnoresYamlAndEnv(t *testing.T) {
	// REQ-CLI-4: saved model wins even when yaml and env provide values.
	store := &fakeModelMemory{models: map[string]string{"coder": "saved"}}
	loader := &fakeModelLoader{
		profiles: map[string]string{"coder": "yaml-model"},
	}

	got := ResolveModel(store, loader, "coder", "env-model")
	if got != "saved" {
		t.Errorf("ResolveModel = %q, want %q (saved always wins)", got, "saved")
	}
}

func TestResolveModelEmptyYamlModelFallsToEnv(t *testing.T) {
	// REQ-CLI-4: empty yaml model string is treated as "no model" — falls
	// through to env, not treated as a valid value.
	store := &fakeModelMemory{models: map[string]string{}}
	loader := &fakeModelLoader{
		profiles: map[string]string{"coder": ""},
	}

	got := ResolveModel(store, loader, "coder", "env-model")
	if got != "env-model" {
		t.Errorf("ResolveModel = %q, want %q (empty yaml model falls to env)", got, "env-model")
	}
}
