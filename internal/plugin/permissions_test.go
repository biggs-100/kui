package plugin

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPluginNewPermissionChecker(t *testing.T) {
	pc := NewPermissionChecker("")
	if pc == nil {
		t.Fatal("NewPermissionChecker returned nil")
	}
	if pc.mode != PermissionModeWarnOnly {
		t.Errorf("default mode = %q, want %q", pc.mode, PermissionModeWarnOnly)
	}
	if pc.rules == nil {
		t.Error("rules map not initialized")
	}
}

func TestPluginNewPermissionCheckerWithFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "permissions.yaml")
	pc := NewPermissionChecker(filePath)
	if pc.filePath != filePath {
		t.Errorf("filePath = %q, want %q", pc.filePath, filePath)
	}
}

func TestPluginCheckUnknown(t *testing.T) {
	pc := NewPermissionChecker("")
	result := pc.Check("my-plugin", "filesystem.read")
	if result != PermissionUnknown {
		t.Errorf("Check unknown permission = %v, want %v", result, PermissionUnknown)
	}
}

func TestPluginGrantAndCheckAllowed(t *testing.T) {
	pc := NewPermissionChecker("")
	if err := pc.Grant("my-plugin", "filesystem.read"); err != nil {
		t.Fatalf("Grant() error = %v", err)
	}
	result := pc.Check("my-plugin", "filesystem.read")
	if result != PermissionAllowed {
		t.Errorf("Check after Grant = %v, want %v", result, PermissionAllowed)
	}
}

func TestPluginDenyAndCheckDenied(t *testing.T) {
	pc := NewPermissionChecker("")
	if err := pc.Deny("my-plugin", "network.request"); err != nil {
		t.Fatalf("Deny() error = %v", err)
	}
	result := pc.Check("my-plugin", "network.request")
	if result != PermissionDenied {
		t.Errorf("Check after Deny = %v, want %v", result, PermissionDenied)
	}
}

func TestPluginGrantDuplicate(t *testing.T) {
	pc := NewPermissionChecker("")
	if err := pc.Grant("my-plugin", "fs.read"); err != nil {
		t.Fatalf("first Grant() error = %v", err)
	}
	// Granting again should be idempotent, not error
	if err := pc.Grant("my-plugin", "fs.read"); err != nil {
		t.Fatalf("duplicate Grant() error = %v", err)
	}
	result := pc.Check("my-plugin", "fs.read")
	if result != PermissionAllowed {
		t.Errorf("Check after duplicate Grant = %v, want %v", result, PermissionAllowed)
	}
}

func TestPluginDenyNotFound(t *testing.T) {
	pc := NewPermissionChecker("")
	// Denying a permission that was never granted should create a denied rule
	if err := pc.Deny("my-plugin", "unknown.perm"); err != nil {
		t.Fatalf("Deny() error = %v", err)
	}
	result := pc.Check("my-plugin", "unknown.perm")
	if result != PermissionDenied {
		t.Errorf("Check after Deny = %v, want %v", result, PermissionDenied)
	}
}

func TestPluginRevokeAllowed(t *testing.T) {
	pc := NewPermissionChecker("")
	if err := pc.Grant("my-plugin", "fs.write"); err != nil {
		t.Fatalf("Grant() error = %v", err)
	}
	if err := pc.Revoke("my-plugin", "fs.write"); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	result := pc.Check("my-plugin", "fs.write")
	if result != PermissionUnknown {
		t.Errorf("Check after Revoke = %v, want %v (should be unknown)", result, PermissionUnknown)
	}
}

func TestPluginRevokeDenied(t *testing.T) {
	pc := NewPermissionChecker("")
	if err := pc.Deny("my-plugin", "net.out"); err != nil {
		t.Fatalf("Deny() error = %v", err)
	}
	if err := pc.Revoke("my-plugin", "net.out"); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	result := pc.Check("my-plugin", "net.out")
	if result != PermissionUnknown {
		t.Errorf("Check after Revoke = %v, want %v", result, PermissionUnknown)
	}
}

func TestPluginRevokeNotFound(t *testing.T) {
	pc := NewPermissionChecker("")
	if err := pc.Revoke("my-plugin", "nonexistent"); err != nil {
		t.Fatalf("Revoke() on unknown should not error, got = %v", err)
	}
}

func TestPluginPermissionsListEmpty(t *testing.T) {
	pc := NewPermissionChecker("")
	perms := pc.List("my-plugin")
	if len(perms) != 0 {
		t.Errorf("List empty = %d perms, want 0", len(perms))
	}
}

func TestPluginPermissionsListMultiple(t *testing.T) {
	pc := NewPermissionChecker("")
	_ = pc.Grant("plugin-a", "fs.read")
	_ = pc.Grant("plugin-a", "fs.write")
	_ = pc.Deny("plugin-a", "net.request")
	_ = pc.Grant("plugin-b", "fs.read")

	perms := pc.List("plugin-a")
	if len(perms) != 3 {
		t.Fatalf("List(plugin-a) = %d perms, want 3", len(perms))
	}

	// Verify permissions are present
	permMap := make(map[string]bool)
	for _, p := range perms {
		key := p.Resource + "." + p.Action
		permMap[key] = true
		if p.Plugin != "plugin-a" {
			t.Errorf("Permission.Plugin = %q, want %q", p.Plugin, "plugin-a")
		}
	}
	if !permMap["fs.read"] {
		t.Error("fs.read permission not found")
	}
	if !permMap["fs.write"] {
		t.Error("fs.write permission not found")
	}
	if !permMap["net.request"] {
		t.Error("net.request permission not found")
	}

	// plugin-b should not appear
	permsB := pc.List("plugin-b")
	if len(permsB) != 1 {
		t.Fatalf("List(plugin-b) = %d perms, want 1", len(permsB))
	}
}

func TestPluginWarnOnlyModeDefault(t *testing.T) {
	pc := NewPermissionChecker("")
	if pc.Mode() != PermissionModeWarnOnly {
		t.Errorf("default Mode = %q, want %q", pc.Mode(), PermissionModeWarnOnly)
	}

	// In warn-only mode, even denied permissions should return a result
	// but the mode check should indicate warn-only behavior
	_ = pc.Deny("my-plugin", "dangerous.action")
	if pc.Mode() != PermissionModeWarnOnly {
		t.Error("mode should still be warn-only after deny")
	}
}

func TestPluginEnforceModeBlocksOnDeny(t *testing.T) {
	pc := NewPermissionChecker("")
	pc.SetMode(PermissionModeEnforce)

	if pc.Mode() != PermissionModeEnforce {
		t.Errorf("Mode = %q, want %q", pc.Mode(), PermissionModeEnforce)
	}

	// In enforce mode, denied permissions should remain denied
	_ = pc.Deny("my-plugin", "dangerous.action")
	result := pc.Check("my-plugin", "dangerous.action")
	if result != PermissionDenied {
		t.Errorf("Check in enforce mode = %v, want %v", result, PermissionDenied)
	}
}

func TestPluginEnforceModeAllowsGranted(t *testing.T) {
	pc := NewPermissionChecker("")
	pc.SetMode(PermissionModeEnforce)

	_ = pc.Grant("my-plugin", "safe.action")
	result := pc.Check("my-plugin", "safe.action")
	if result != PermissionAllowed {
		t.Errorf("Check granted in enforce mode = %v, want %v", result, PermissionAllowed)
	}
}

func TestPluginPermissionsPersistence(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "permissions.yaml")

	// Create and save
	pc := NewPermissionChecker(filePath)
	_ = pc.Grant("my-plugin", "fs.read")
	_ = pc.Grant("my-plugin", "fs.write")
	_ = pc.Deny("my-plugin", "net.request")

	if err := pc.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("permissions file was not created")
	}

	// Load into new checker
	pc2 := NewPermissionChecker(filePath)
	if err := pc2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify loaded state
	if pc2.Check("my-plugin", "fs.read") != PermissionAllowed {
		t.Error("fs.read should be allowed after load")
	}
	if pc2.Check("my-plugin", "fs.write") != PermissionAllowed {
		t.Error("fs.write should be allowed after load")
	}
	if pc2.Check("my-plugin", "net.request") != PermissionDenied {
		t.Error("net.request should be denied after load")
	}
	if pc2.Check("my-plugin", "unknown") != PermissionUnknown {
		t.Error("unknown permission should remain unknown after load")
	}
}

func TestPluginPermissionsLoadNonexistentFile(t *testing.T) {
	pc := NewPermissionChecker("/nonexistent/path/permissions.yaml")
	if err := pc.Load(); err != nil {
		t.Fatalf("Load() on nonexistent file should not error, got = %v", err)
	}
	// Should still be in default state
	result := pc.Check("any-plugin", "any.perm")
	if result != PermissionUnknown {
		t.Errorf("Check after load nonexistent = %v, want %v", result, PermissionUnknown)
	}
}

func TestPluginPermissionsGrantTimestamp(t *testing.T) {
	pc := NewPermissionChecker("")
	before := time.Now()
	_ = pc.Grant("my-plugin", "test.perm")
	after := time.Now()

	perms := pc.List("my-plugin")
	if len(perms) != 1 {
		t.Fatalf("List = %d perms, want 1", len(perms))
	}
	if perms[0].GrantedAt.Before(before) || perms[0].GrantedAt.After(after) {
		t.Errorf("GrantedAt = %v, should be between %v and %v", perms[0].GrantedAt, before, after)
	}
}

func TestPluginPermissionsMultiplePlugins(t *testing.T) {
	pc := NewPermissionChecker("")
	_ = pc.Grant("plugin-a", "perm1")
	_ = pc.Grant("plugin-b", "perm2")
	_ = pc.Grant("plugin-a", "perm3")

	// plugin-a should have 2 permissions
	permsA := pc.List("plugin-a")
	if len(permsA) != 2 {
		t.Errorf("List(plugin-a) = %d perms, want 2", len(permsA))
	}

	// plugin-b should have 1 permission
	permsB := pc.List("plugin-b")
	if len(permsB) != 1 {
		t.Errorf("List(plugin-b) = %d perms, want 1", len(permsB))
	}

	// Permissions are scoped per plugin
	if pc.Check("plugin-b", "perm1") != PermissionUnknown {
		t.Error("plugin-b should not have perm1")
	}
	if pc.Check("plugin-a", "perm2") != PermissionUnknown {
		t.Error("plugin-a should not have perm2")
	}
}
