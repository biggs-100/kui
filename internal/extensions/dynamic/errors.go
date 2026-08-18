package dynamic

import "fmt"

// ManifestError wraps failures parsing or loading an extension.yaml manifest.
// It carries the file path and optional field for clear diagnostics.
type ManifestError struct {
	File  string
	Field string
	Err   error
}

func (e *ManifestError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("manifest %q: field %q: %v", e.File, e.Field, e.Err)
	}
	return fmt.Sprintf("manifest %q: %v", e.File, e.Err)
}

// Unwrap exposes the underlying cause to errors.Is / errors.As.
func (e *ManifestError) Unwrap() error {
	return e.Err
}

// ProtocolError wraps JSON-RPC protocol failures during extension
// communication. It carries the extension name and method for traceability.
type ProtocolError struct {
	Extension string
	Method    string
	Err       error
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("extension %q: protocol error on %q: %v", e.Extension, e.Method, e.Err)
}

// Unwrap exposes the underlying cause to errors.Is / errors.As.
func (e *ProtocolError) Unwrap() error {
	return e.Err
}

// SpawnError wraps subprocess spawn failures. It carries the extension name
// and entry point for clear diagnostics.
type SpawnError struct {
	Extension  string
	EntryPoint string
	Err        error
}

func (e *SpawnError) Error() string {
	return fmt.Sprintf("extension %q: spawn %q failed: %v", e.Extension, e.EntryPoint, e.Err)
}

// Unwrap exposes the underlying cause to errors.Is / errors.As.
func (e *SpawnError) Unwrap() error {
	return e.Err
}
