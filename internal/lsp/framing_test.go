package lsp

import (
	"bytes"
	"io"
	"testing"
)

func TestWriteAndReadMessage(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty body", []byte{}},
		{"simple json", []byte(`{"jsonrpc":"2.0","id":1}`)},
		{"larger payload", []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteMessage(&buf, tt.data)
			if err != nil {
				t.Fatalf("WriteMessage() error: %v", err)
			}

			got, err := ReadMessage(&buf)
			if err != nil {
				t.Fatalf("ReadMessage() error: %v", err)
			}

			if !bytes.Equal(got, tt.data) {
				t.Errorf("round-trip mismatch: got %q, want %q", got, tt.data)
			}
		})
	}
}

func TestReadMessageInvalidContentLength(t *testing.T) {
	// Content-Length with non-numeric value
	input := "Content-Length: abc\r\n\r\n"
	_, err := ReadMessage(bytes.NewBufferString(input))
	if err == nil {
		t.Fatal("expected error for invalid Content-Length, got nil")
	}
}

func TestReadMessageMissingContentLength(t *testing.T) {
	// Headers without Content-Length
	input := "Content-Type: application/json\r\n\r\n"
	_, err := ReadMessage(bytes.NewBufferString(input))
	if err == nil {
		t.Fatal("expected error for missing Content-Length, got nil")
	}
}

func TestReadMessageIncomplete(t *testing.T) {
	// Content-Length says 100 but body is shorter
	input := "Content-Length: 100\r\n\r\nshort"
	_, err := ReadMessage(bytes.NewBufferString(input))
	if err == nil {
		t.Fatal("expected error for incomplete body, got nil")
	}
}

func TestReadMessageFromPipe(t *testing.T) {
	// Verify framing works across io.Pipe (simulating real stdio)
	pr, pw := io.Pipe()

	data := []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`)

	go func() {
		err := WriteMessage(pw, data)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Close()
	}()

	got, err := ReadMessage(pr)
	if err != nil {
		t.Fatalf("ReadMessage() from pipe error: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Errorf("pipe round-trip mismatch: got %q, want %q", got, data)
	}
}

func TestMultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	msg1 := []byte(`{"jsonrpc":"2.0","id":1}`)
	msg2 := []byte(`{"jsonrpc":"2.0","id":2,"method":"test"}`)

	if err := WriteMessage(&buf, msg1); err != nil {
		t.Fatalf("WriteMessage(1) error: %v", err)
	}
	if err := WriteMessage(&buf, msg2); err != nil {
		t.Fatalf("WriteMessage(2) error: %v", err)
	}

	got1, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage(1) error: %v", err)
	}
	if !bytes.Equal(got1, msg1) {
		t.Errorf("message 1: got %q, want %q", got1, msg1)
	}

	got2, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage(2) error: %v", err)
	}
	if !bytes.Equal(got2, msg2) {
		t.Errorf("message 2: got %q, want %q", got2, msg2)
	}
}
