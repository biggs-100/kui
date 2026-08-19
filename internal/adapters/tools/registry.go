package tools

import (
	"time"

	"github.com/biggs-100/kui/internal/core"
)

// Default returns the built-in tool set in advertisement order: read_file,
// write_file, bash (REQ-TOOLS-4). File tools are confined to root; bashTimeout
// bounds every command (zero selects the default of 30s).
func Default(root string, bashTimeout time.Duration) []core.Tool {
	return DefaultWithSyncer(root, bashTimeout, nil)
}

// DefaultWithSyncer returns the built-in tool set with optional LSP file sync.
// When syncer is non-nil, read_file and write_file tools send DidOpen/DidChange
// notifications to the LSP server via the syncer.
func DefaultWithSyncer(root string, bashTimeout time.Duration, syncer FileSyncer) []core.Tool {
	var rf, wf core.Tool
	if syncer != nil {
		rf = NewReadFileWithSync(root, syncer)
		wf = NewWriteFileWithSync(root, syncer)
	} else {
		rf = NewReadFile(root)
		wf = NewWriteFile(root)
	}
	return []core.Tool{
		rf,
		wf,
		NewBash(bashTimeout),
	}
}
