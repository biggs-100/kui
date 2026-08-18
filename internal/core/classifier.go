package core

import "strings"

// profileMarker is the content substring that identifies a profile-switch
// metadata message. Such messages are always protected from compaction.
const profileMarker = "Profile switched"

// ClassifyMessages tags messages as protected or compactable.
// Rules:
//   - RoleSystem messages are protected
//   - Messages containing "Profile switched" in content are protected
//   - Tool result messages whose matching tool call is protected are also protected
//   - All other messages are compactable
func ClassifyMessages(messages []Message) ([]Message, []Message) {
	// First pass: classify by role and profile marker.
	protected := make([]Message, 0)
	compactable := make([]Message, 0)

	for _, m := range messages {
		if m.Role == RoleSystem || strings.Contains(m.Content, profileMarker) {
			protected = append(protected, m)
		} else {
			compactable = append(compactable, m)
		}
	}

	// Collect tool call IDs from protected assistant messages so their
	// matching tool results are also preserved atomically.
	protectedToolCalls := make(map[string]bool, len(protected))
	for _, m := range protected {
		if m.ToolCall != nil {
			protectedToolCalls[m.ToolCall.ID] = true
		}
	}

	// Second pass: move tool results with matching protected call IDs
	// from compactable to protected.
	if len(protectedToolCalls) > 0 {
		stillCompactable := make([]Message, 0, len(compactable))
		for _, m := range compactable {
			if m.ToolCallID != "" && protectedToolCalls[m.ToolCallID] {
				protected = append(protected, m)
			} else {
				stillCompactable = append(stillCompactable, m)
			}
		}
		compactable = stillCompactable
	}

	return protected, compactable
}
