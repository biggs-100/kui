package tools

import (
	"time"

	"github.com/biggs-100/kui/internal/core"
)

// Default returns the built-in tool set in advertisement order: read_file,
// write_file, bash (REQ-TOOLS-4). File tools are confined to root; bashTimeout
// bounds every command (zero selects the default of 30s).
func Default(root string, bashTimeout time.Duration) []core.Tool {
	return []core.Tool{
		NewReadFile(root),
		NewWriteFile(root),
		NewBash(bashTimeout),
	}
}
