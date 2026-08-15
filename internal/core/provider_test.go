package core

import "testing"

// TestRoleSystemMatchesWireFormat pins the system-role constant to the role
// string every OpenAI-compatible provider serializes verbatim (the adapter
// maps Message.Role through unchanged, see adapters/providers/openai). The
// profile-context marker must arrive at the provider with role "system"
// (D16, REQ-LOOP-6).
func TestRoleSystemMatchesWireFormat(t *testing.T) {
	if RoleSystem != "system" {
		t.Errorf("RoleSystem = %q, want %q", RoleSystem, "system")
	}
}
