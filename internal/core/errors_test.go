package core

import (
	"errors"
	"strings"
	"testing"
)

// TestErrorTypesReportContext verifies each profile-system error names the
// entity that failed, so CLI and log output can point at the exact cause
// (D16, REQ-PERM-4, REQ-SKILL-3, REQ-PROFILE-4).
func TestErrorTypesReportContext(t *testing.T) {
	cause := errors.New("root cause")

	tests := []struct {
		name    string
		err     error
		wantErr string // substring the message must contain
	}{
		{"unknown profile", &UnknownProfileError{Name: "writer"}, `unknown profile "writer"`},
		{"permission", &PermissionError{Tool: "bash"}, `tool "bash"`},
		{"profile activation names file", &ProfileActivationError{Name: "writer", File: "SYSTEM.md", Err: cause}, "SYSTEM.md"},
		{"skill load names file", &SkillLoadError{Name: "go", File: "skill.yaml", Err: cause}, "skill.yaml"},
		{"store op", &StoreError{Op: "save", Err: cause}, "save"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); !strings.Contains(got, tt.wantErr) {
				t.Errorf("Error() = %q, want it to contain %q", got, tt.wantErr)
			}
		})
	}
}

// TestErrorTypesUnwrapRootCause verifies the wrapping errors expose their
// underlying cause to errors.Is / errors.As, while non-wrapping errors stay
// opaque — matching the established UnknownToolError/ToolError contract.
func TestErrorTypesUnwrapRootCause(t *testing.T) {
	cause := errors.New("root cause")

	for _, err := range []error{
		&ProfileActivationError{Name: "writer", Err: cause},
		&SkillLoadError{Name: "go", Err: cause},
		&StoreError{Op: "save", Err: cause},
	} {
		if !errors.Is(err, cause) {
			t.Errorf("%T does not unwrap to %v", err, cause)
		}
	}

	if errors.Is(&UnknownProfileError{Name: "x"}, cause) {
		t.Error("UnknownProfileError must not unwrap a root cause")
	}
	if errors.Is(&PermissionError{Tool: "bash"}, cause) {
		t.Error("PermissionError must not unwrap a root cause")
	}
}
