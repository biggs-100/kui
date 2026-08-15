package core

import (
	"context"
	"encoding/json"
)

// Tool is the port every built-in or external tool implements (D3). Schema
// returns the raw JSON parameter schema used to advertise the tool to the
// provider. Execute receives the raw JSON arguments and returns the tool
// result text.
type Tool interface {
	Name() string
	Description() string
	Schema() string
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// Registry keeps tools in registration order for stable advertisement and
// resolves names for dispatch (D4).
type Registry struct {
	order  []string
	byName map[string]Tool
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]Tool)}
}

// Register adds a tool. Duplicate names are rejected to keep the lookup map
// unambiguous.
func (r *Registry) Register(tool Tool) error {
	name := tool.Name()
	if _, exists := r.byName[name]; exists {
		return &DuplicateToolError{Name: name}
	}
	r.byName[name] = tool
	r.order = append(r.order, name)
	return nil
}

// Get resolves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	tool, ok := r.byName[name]
	return tool, ok
}

// List returns the registered tools in registration order.
func (r *Registry) List() []Tool {
	tools := make([]Tool, 0, len(r.order))
	for _, name := range r.order {
		tools = append(tools, r.byName[name])
	}
	return tools
}
