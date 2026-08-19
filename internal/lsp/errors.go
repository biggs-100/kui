package lsp

import "fmt"

// LspError represents an error returned by an LSP server.
type LspError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Standard LSP error codes.
const (
	// ServerNotInitialized is returned when a request is made before initialization.
	ServerNotInitialized = -32002
	// RequestFailed is a general request failure.
	RequestFailed = -32803
	// ServerCancelled is returned when the server cancels a request.
	ServerCancelled = -32802
	// ContentModified is returned when content changed before a response was sent.
	ContentModified = -32801
	// RequestCancelled is returned when a request is cancelled by the client.
	RequestCancelled = -32800
)

// Error implements the error interface for LspError.
func (e *LspError) Error() string {
	return fmt.Sprintf("lsp error %d: %s", e.Code, e.Message)
}

// LspConnectionError wraps failures connecting to or communicating with an
// LSP server subprocess.
type LspConnectionError struct {
	Server string
	Err    error
}

func (e *LspConnectionError) Error() string {
	return fmt.Sprintf("lsp server %q connection failed: %v", e.Server, e.Err)
}

// Unwrap exposes the underlying cause to errors.Is / errors.As.
func (e *LspConnectionError) Unwrap() error {
	return e.Err
}

// ServerNotReadyError is returned when an LSP tool is called before the server is running.
type ServerNotReadyError struct {
	Tool string
}

func (e *ServerNotReadyError) Error() string {
	return fmt.Sprintf("lsp tool %q: server not ready", e.Tool)
}
