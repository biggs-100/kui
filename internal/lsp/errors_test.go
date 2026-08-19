package lsp

import (
	"errors"
	"testing"
)

func TestLspErrorString(t *testing.T) {
	tests := []struct {
		name string
		err  *LspError
		want string
	}{
		{
			name: "server not initialized",
			err:  &LspError{Code: ServerNotInitialized, Message: "server not initialized"},
			want: "lsp error -32002: server not initialized",
		},
		{
			name: "request failed",
			err:  &LspError{Code: RequestFailed, Message: "request failed"},
			want: "lsp error -32803: request failed",
		},
		{
			name: "server cancelled",
			err:  &LspError{Code: ServerCancelled, Message: "server cancelled request"},
			want: "lsp error -32802: server cancelled request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLspConnectionErrorString(t *testing.T) {
	err := &LspConnectionError{
		Server: "gopls",
		Err:    errors.New("process exited"),
	}

	want := `lsp server "gopls" connection failed: process exited`
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestLspConnectionErrorUnwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := &LspConnectionError{Server: "gopls", Err: inner}

	if !errors.Is(err, inner) {
		t.Error("errors.Is should find inner error via Unwrap")
	}
}

func TestServerNotReadyErrorString(t *testing.T) {
	err := &ServerNotReadyError{Tool: "lsp_hover"}
	want := `lsp tool "lsp_hover": server not ready`
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestLspErrorConstants(t *testing.T) {
	if ServerNotInitialized != -32002 {
		t.Errorf("ServerNotInitialized = %d, want -32002", ServerNotInitialized)
	}
	if RequestFailed != -32803 {
		t.Errorf("RequestFailed = %d, want -32803", RequestFailed)
	}
	if ServerCancelled != -32802 {
		t.Errorf("ServerCancelled = %d, want -32802", ServerCancelled)
	}
	if ContentModified != -32801 {
		t.Errorf("ContentModified = %d, want -32801", ContentModified)
	}
	if RequestCancelled != -32800 {
		t.Errorf("RequestCancelled = %d, want -32800", RequestCancelled)
	}
}
