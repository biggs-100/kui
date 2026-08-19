package plugin

import "fmt"

// ManifestError wraps failures parsing or loading a plugin manifest.
type ManifestError struct {
	File  string
	Field string
	Err   error
}

func (e *ManifestError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("plugin manifest %q: field %q: %v", e.File, e.Field, e.Err)
	}
	return fmt.Sprintf("plugin manifest %q: %v", e.File, e.Err)
}

func (e *ManifestError) Unwrap() error {
	return e.Err
}

// PermissionError represents a plugin permission denial or unknown state.
type PermissionError struct {
	Plugin  string
	Resource string
	Action  string
	Message string
}

func (e *PermissionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("permission denied for plugin %q on %s.%s: %s", e.Plugin, e.Resource, e.Action, e.Message)
	}
	return fmt.Sprintf("permission denied for plugin %q on %s.%s", e.Plugin, e.Resource, e.Action)
}

// NotFoundError represents a missing plugin or resource.
type NotFoundError struct {
	Name string
	Type string
}

func (e *NotFoundError) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("%s %q not found", e.Type, e.Name)
	}
	return fmt.Sprintf("%q not found", e.Name)
}
