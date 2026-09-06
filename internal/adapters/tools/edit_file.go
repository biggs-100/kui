package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// utf8BOM is preserved across edits: the model never includes it in old_text,
// so it is stripped before matching and restored before writing.
const utf8BOM = "\xEF\xBB\xBF"

// EditFile replaces exactly one text block in an existing file inside the
// workspace root (REQ-TOOLS-8). The match is byte-exact, whitespace included,
// over content normalized to LF; CRLF endings and a BOM are restored on
// write. Writes are atomic (temp file in the same directory plus rename).
// Paths escaping the root are rejected before any I/O (D11).
type EditFile struct {
	root   string
	syncer FileSyncer // optional — nil disables LSP notifications
}

// NewEditFile returns an edit_file tool confined to root.
func NewEditFile(root string) *EditFile {
	return &EditFile{root: root}
}

// NewEditFileWithSync returns an edit_file tool with LSP file sync support.
func NewEditFileWithSync(root string, syncer FileSyncer) *EditFile {
	return &EditFile{root: root, syncer: syncer}
}

// Name returns the stable tool name (REQ-TOOLS-4).
func (t *EditFile) Name() string { return "edit_file" }

// Description returns the tool description (REQ-TOOLS-4).
func (t *EditFile) Description() string {
	return "Replace one exact text block in a file inside the workspace; read the file first with read_file and copy old_text exactly including whitespace"
}

// Schema returns the raw JSON parameter schema (D3, REQ-TOOLS-4).
func (t *EditFile) Schema() string {
	return `{"type":"object","properties":{"path":{"type":"string"},"old_text":{"type":"string"},"new_text":{"type":"string"},"occurrence":{"type":"integer","minimum":1}},"required":["path","old_text","new_text"]}`
}

// Execute replaces one block in the file at path. Without occurrence the
// old_text must match exactly once; with occurrence (1-based) the Nth match
// is replaced. The result carries a unified diff so transcript views can
// highlight it without extra wiring.
func (t *EditFile) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path       string `json:"path"`
		OldText    string `json:"old_text"`
		NewText    string `json:"new_text"`
		Occurrence *int   `json:"occurrence"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if in.Path == "" {
		return "", errors.New("path must not be empty")
	}
	if in.OldText == "" {
		return "", errors.New("old_text must not be empty")
	}
	// *int distinguishes an absent occurrence (unique-match mode) from an
	// explicit occurrence:0, which the schema rejects (minimum 1).
	if in.Occurrence != nil && *in.Occurrence < 1 {
		return "", fmt.Errorf("occurrence must be >= 1, got %d", *in.Occurrence)
	}
	occurrence := 0
	if in.Occurrence != nil {
		occurrence = *in.Occurrence
	}
	resolved, err := resolvePath(t.root, in.Path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		// Wrap with the caller-supplied relative path: stable across
		// platforms, unlike the absolute resolved path in the raw error.
		return "", fmt.Errorf("cannot read %q: %w", in.Path, err)
	}

	// Binary guard: same criterion as grep.isBinary — NUL in first 512 bytes.
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return "", fmt.Errorf("cannot edit binary file %q", in.Path)
	}

	// Strip BOM for matching; the model never sends it. Endings are
	// normalized to LF so callers can express old_text with plain \n.
	raw := string(data)
	bom := ""
	if strings.HasPrefix(raw, utf8BOM) {
		bom = utf8BOM
		raw = strings.TrimPrefix(raw, utf8BOM)
	}
	crlf := dominantCRLF(raw)
	base := normalizeLF(raw)
	old := normalizeLF(in.OldText)
	new := normalizeLF(in.NewText)

	if old == new {
		return "", errors.New("old_text and new_text are identical (no-op)")
	}
	matches := strings.Count(base, old)
	switch {
	case matches == 0:
		return "", fmt.Errorf("no match for old_text in %q (0 matches); read the file with read_file and copy the exact text including whitespace", in.Path)
	case occurrence == 0 && matches > 1:
		return "", fmt.Errorf("old_text matches %d regions in %q; add more surrounding context or pass occurrence (1..%d)", matches, in.Path, matches)
	case occurrence > matches:
		return "", fmt.Errorf("occurrence %d out of range: old_text matches %d region(s) in %q", occurrence, matches, in.Path)
	}

	updated := base
	if occurrence == 0 {
		updated = strings.Replace(base, old, new, 1)
	} else {
		updated = replaceNth(base, old, new, occurrence)
	}

	final := bom + restoreCRLF(updated, crlf)
	if err := atomicWrite(resolved, []byte(final)); err != nil {
		return "", err
	}

	// LSP file sync: notify server that file changed (best-effort).
	if t.syncer != nil {
		uri := pathToFileURI(resolved)
		_ = t.syncer.DidChange(uri, final)
	}

	return fmt.Sprintf("Successfully replaced 1 block(s) in %s.\n\n%s", in.Path, unifiedDiff(in.Path, base, updated)), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// helpers
// ──────────────────────────────────────────────────────────────────────────────

// normalizeLF folds CRLF and lone CR to LF for matching.
func normalizeLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// dominantCRLF reports whether CRLF is the dominant line ending. Ties and
// LF-only content resolve to LF; mixed files are unified to the dominant
// ending on write (documented behavior).
func dominantCRLF(s string) bool {
	crlf := strings.Count(s, "\r\n")
	return crlf > strings.Count(s, "\n")-crlf
}

// restoreCRLF maps LF back to CRLF when the original file was CRLF-dominant.
func restoreCRLF(s string, crlf bool) string {
	if !crlf {
		return s
	}
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// replaceNth replaces the nth (1-based) non-overlapping occurrence of old.
// Callers must guarantee 1 <= n <= strings.Count(s, old).
func replaceNth(s, old, new string, n int) string {
	idx := -1
	from := 0
	for i := 1; i <= n; i++ {
		j := strings.Index(s[from:], old)
		if j < 0 {
			return s
		}
		idx = from + j
		from = idx + len(old)
	}
	return s[:idx] + new + s[idx+len(old):]
}

// atomicWrite writes data via a temp file in the same directory plus rename,
// so a crash never leaves a half-written file behind.
// NOTE: the temp file mode is normalized to 0644 on every write; the
// original file mode is NOT preserved (documented behavior).
func atomicWrite(target string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".edit-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup: after a successful rename the name no longer
	// exists and Remove is a no-op.
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, target)
}

// unifiedDiff builds a minimal unified diff with 3 lines of context, enough
// for transcript views keying off the "diff --git" header.
func unifiedDiff(displayPath, base, updated string) string {
	a := strings.Split(base, "\n")
	b := strings.Split(updated, "\n")
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	const ctx = 3
	start := prefix - ctx
	if start < 0 {
		start = 0
	}
	endA := len(a) - suffix + ctx
	if endA > len(a) {
		endA = len(a)
	}
	endB := len(b) - suffix + ctx
	if endB > len(b) {
		endB = len(b)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "diff --git a/%s b/%s\n", displayPath, displayPath)
	fmt.Fprintf(&sb, "--- a/%s\n", displayPath)
	fmt.Fprintf(&sb, "+++ b/%s\n", displayPath)
	fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n", start+1, endA-start, start+1, endB-start)
	for _, l := range a[start:prefix] {
		sb.WriteString(" " + l + "\n")
	}
	for _, l := range a[prefix : len(a)-suffix] {
		sb.WriteString("-" + l + "\n")
	}
	for _, l := range b[prefix : len(b)-suffix] {
		sb.WriteString("+" + l + "\n")
	}
	for _, l := range a[len(a)-suffix : endA] {
		sb.WriteString(" " + l + "\n")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}
