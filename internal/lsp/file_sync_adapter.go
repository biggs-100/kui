package lsp

import "github.com/biggs-100/kui/internal/adapters/tools"

// LspFileSyncer adapts LspManager to the tools.FileSyncer interface.
// It delegates DidOpen/DidChange to the LspClient for the given root URI.
// This bridges LSP file synchronization with the file tools (read_file,
// write_file) so that LSP servers receive didOpen/didChange notifications.
type LspFileSyncer struct {
	mgr *LspManager
}

// NewLspFileSyncer creates an adapter that implements tools.FileSyncer
// by delegating to the LspManager's client for the workspace root.
func NewLspFileSyncer(mgr *LspManager) *LspFileSyncer {
	return &LspFileSyncer{mgr: mgr}
}

// Compile-time check: *LspFileSyncer must satisfy tools.FileSyncer.
var _ tools.FileSyncer = (*LspFileSyncer)(nil)

// DidOpen notifies the LSP server that a file was opened. It resolves the
// client from the manager using the URI as the root key. Errors from GetServer
// are silently ignored to maintain graceful degradation.
func (s *LspFileSyncer) DidOpen(uri, languageID, content string) error {
	client, err := s.mgr.GetServer(uri)
	if err != nil {
		return nil // graceful degradation: file sync failure should not block the read
	}
	return client.DidOpen(uri, languageID, 1, content)
}

// DidChange notifies the LSP server that a file's content changed.
func (s *LspFileSyncer) DidChange(uri, content string) error {
	client, err := s.mgr.GetServer(uri)
	if err != nil {
		return nil // graceful degradation
	}
	return client.DidChange(uri, 1, []TextDocumentContentChangeEvent{
		{Text: content},
	})
}
