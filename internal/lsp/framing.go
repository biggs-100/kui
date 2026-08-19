package lsp

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// HeaderContentLength is the LSP Content-Length header name.
const HeaderContentLength = "Content-Length"

// ReadMessage reads a Content-Length framed message from the reader.
// Format: "Content-Length: N\r\n\r\n" followed by N bytes of JSON.
func ReadMessage(r io.Reader) ([]byte, error) {
	// Read headers until blank line
	contentLength := -1
	buf := make([]byte, 0, 256)

	for {
		// Read one byte at a time to find \r\n\r\n
		b := make([]byte, 1)
		for {
			n, err := r.Read(b)
			if n == 0 {
				if err != nil {
					return nil, fmt.Errorf("read header: %w", err)
				}
				continue
			}
			buf = append(buf, b[0])
			break
		}

		// Check if we've hit the end of headers (\r\n\r\n)
		if len(buf) >= 4 && string(buf[len(buf)-4:]) == "\r\n\r\n" {
			break
		}

		// Safety limit
		if len(buf) > 8192 {
			return nil, fmt.Errorf("headers too large (>%d bytes)", 8192)
		}
	}

	// Parse Content-Length from headers
	headerBlock := string(buf)
	for _, line := range strings.Split(headerBlock, "\r\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == HeaderContentLength {
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length %q: %w", val, err)
			}
			contentLength = n
		}
	}

	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	if contentLength == 0 {
		return []byte{}, nil
	}

	// Read exactly contentLength bytes
	body := make([]byte, contentLength)
	_, err := io.ReadFull(r, body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}

// WriteMessage writes a Content-Length framed message to the writer.
func WriteMessage(w io.Writer, data []byte) error {
	header := fmt.Sprintf("%s: %d\r\n\r\n", HeaderContentLength, len(data))
	if _, err := w.Write([]byte(header)); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	return nil
}
