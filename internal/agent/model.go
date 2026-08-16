package agent

// defaultModel is the terminal fallback of the REQ-CLI-4 resolution chain
// when no saved model, no profile.yaml model, and no OPENAI_MODEL are present.
const defaultModel = "gpt-4o-mini"

// ModelMemory is the port for reading the saved per-profile model
// (REQ-CLI-4, REQ-PROFILE-4). The concrete store.Store satisfies this
// through Go's structural typing.
type ModelMemory interface {
	Get(profile string) (string, bool)
}

// ResolvedProfile is the subset of a profile that ResolveModel needs from
// the layered YAML loader.
type ResolvedProfile struct {
	Model string
}

// ModelLoader is the port for resolving a profile's YAML config
// (REQ-CLI-4, REQ-PROFILE-2). The concrete profile.Loader satisfies this
// through Go's structural typing.
type ModelLoader interface {
	Resolve(name string) (*ResolvedProfile, error)
}

// ResolveModel applies the REQ-CLI-4 resolution chain: the profile's saved
// model (ModelMemory), then the layered profile.yaml model (ModelLoader),
// then envModel (the caller reads OPENAI_MODEL), then the built-in default.
// It is public so the TUI controller can reuse the same chain without
// duplicating logic. envModel is passed in rather than read here so the
// agent package stays free of os imports (guard test).
func ResolveModel(store ModelMemory, loader ModelLoader, name string, envModel string) string {
	if model, ok := store.Get(name); ok {
		return model
	}
	if resolved, err := loader.Resolve(name); err == nil && resolved.Model != "" {
		return resolved.Model
	}
	if envModel != "" {
		return envModel
	}
	return defaultModel
}
