package core

// PermissionEvaluator gates tool advertisement and dispatch for the active
// profile (D15, REQ-PERM-3/4). Filter narrows the advertised tool set before
// Chat so the provider never sees denied tools; Allow guards a single tool
// dispatch at execution time.
type PermissionEvaluator interface {
	Allow(name string) bool
	Filter(tools []Tool) []Tool
}
