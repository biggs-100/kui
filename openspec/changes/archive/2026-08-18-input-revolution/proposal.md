# Proposal: Input Revolution

## Intent

kui's input is a raw `string` — append-only, single-line, no cursor movement, no history, no autocomplete. OpenCode has a full `TextareaRenderable` with multi-line editing, cursor navigation, selection, undo/redo, clipboard paste, autocomplete, and history. This change replaces kui's string input with a proper text editor widget using `bubbles/textarea`.

## Current Gap

| Feature | kui | OpenCode |
|---------|-----|----------|
| Text engine | Raw `string` | `TextareaRenderable` |
| Multi-line | ❌ | ✅ |
| Cursor movement | ❌ (append-only) | ✅ |
| Selection | ❌ | ✅ |
| Undo/Redo | ❌ | ✅ |
| Clipboard/Paste | ❌ | ✅ |
| Input history | ❌ | ✅ (JSONL, 50 entries) |
| Autocomplete | ❌ | ✅ (`@` files + `/` commands) |
| Slash commands | 5 hardcoded | 20+ with fuzzy search |
| Keybindings | 5 hardcoded | Fully configurable |
| Prompt stash | ❌ | ✅ |

## Proposed Solution

### Phase 1A: Replace string with bubbles/textarea
- Replace `App.input string` with `textarea.Model`
- Integrate into Bubble Tea Update/View cycle
- Preserve existing keybindings (Tab, Enter, q, Ctrl+C)
- Add cursor movement (arrows, home/end)
- Add word navigation (Ctrl+Left/Right)
- Add selection (Shift+arrows)
- Add undo/redo (Ctrl+Z/Y)

### Phase 1B: Input history
- Store prompts in JSONL file (`~/.config/kui/history.jsonl`)
- Navigate with up/down arrows (when cursor at start/end)
- 50 entry limit with auto-trim
- Duplicate suppression

### Phase 1C: Slash command autocomplete
- When user types `/` at start, show matching commands
- Popup/dropdown with fuzzy matching
- Up/down to navigate, Enter/Tab to select, Escape to dismiss
- Commands: `/reload`, `/sessions`, `/resume`, `/quit`, `/help`, `/theme`, `/status`, `/clear`

### Phase 1D: Clipboard paste
- Bracketed paste detection
- Smart paste: file paths → file content, large text → summary
- Ctrl+V paste support

## Scope

### In Scope
1. Replace string input with `textarea.Model`
2. Multi-line editing (Shift+Enter for newline)
3. Cursor movement (arrows, home/end, word jumps)
4. Selection (Shift+arrows, Ctrl+A)
5. Undo/redo (Ctrl+Z/Y)
6. Input history (up/down arrows, JSONL persistence)
7. Slash command autocomplete (popup with fuzzy match)
8. Clipboard paste (bracketed paste protocol)

### Out of Scope
1. File/agent mention autocomplete (`@`) — Phase 2
2. Prompt stash — Phase 2
3. Shell mode (`!` prefix) — Phase 2
4. Configurable keybindings — Phase 2
5. External editor integration — Phase 2
6. Syntax highlighting in input — Phase 2

## Success Criteria

1. Multi-line input works (Shift+Enter creates newline)
2. Cursor moves left/right/home/end
3. Words navigate with Ctrl+Left/Right
4. Selection works with Shift+arrows
5. Undo/redo works with Ctrl+Z/Y
6. Up/down arrows navigate history
7. `/` triggers autocomplete popup
8. Ctrl+V pastes from clipboard
9. All existing keybindings still work (Tab, Enter, q, Ctrl+C)

## Risks

1. **Bubble Tea paste support**: Not all terminals support bracketed paste
   - Mitigation: Graceful degradation, document limitation
2. **Performance**: textarea.Model may be slower than raw string
   - Mitigation: Profile, optimize if needed
3. **Keybinding conflicts**: textarea.Model uses some keys we need
   - Mitigation: Careful keymap configuration, override where needed
