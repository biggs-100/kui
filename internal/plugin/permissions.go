package plugin

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// PermissionResult represents the outcome of a permission check.
type PermissionResult int

const (
	PermissionAllowed PermissionResult = iota
	PermissionDenied
	PermissionUnknown
)

// PermissionMode controls how denied permissions are handled.
type PermissionMode string

const (
	// PermissionModeWarnOnly logs warnings but allows operations to proceed.
	PermissionModeWarnOnly PermissionMode = "warn-only"
	// PermissionModeEnforce blocks operations when permissions are denied.
	PermissionModeEnforce PermissionMode = "enforce"
)

// Permission represents a single permission entry for a plugin.
type Permission struct {
	Plugin    string    `yaml:"plugin"`
	Resource  string    `yaml:"resource"`
	Action    string    `yaml:"action"`
	Granted   bool      `yaml:"granted"`
	GrantedAt time.Time `yaml:"granted_at"`
}

// permissionRule is the internal representation of a permission decision.
type permissionRule struct {
	granted   bool
	grantedAt time.Time
}

// PermissionChecker manages plugin permissions with support for
// warn-only and enforce modes, and persistence to a YAML file.
type PermissionChecker struct {
	rules          map[string]map[string]*permissionRule // plugin -> permission -> rule
	mode           PermissionMode
	filePath       string
	consentPrompt  *ConsentPrompt
	mu             sync.RWMutex
}

// NewPermissionChecker creates a new PermissionChecker.
// If filePath is non-empty, permissions can be persisted to and loaded from that file.
// The default mode is warn-only.
func NewPermissionChecker(filePath string) *PermissionChecker {
	return &PermissionChecker{
		rules:    make(map[string]map[string]*permissionRule),
		mode:     PermissionModeWarnOnly,
		filePath: filePath,
	}
}

// Mode returns the current permission mode.
func (pc *PermissionChecker) Mode() PermissionMode {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.mode
}

// SetMode changes the permission enforcement mode.
func (pc *PermissionChecker) SetMode(mode PermissionMode) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.mode = mode
}

// SetConsentPrompt sets the consent prompt for interactive permission requests.
func (pc *PermissionChecker) SetConsentPrompt(prompt *ConsentPrompt) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.consentPrompt = prompt
}

// Check returns the permission result for a given plugin and permission.
// The permission string is treated as "resource.action" format.
func (pc *PermissionChecker) Check(plugin, permission string) PermissionResult {
	pc.mu.RLock()
	pluginRules, ok := pc.rules[plugin]
	if !ok {
		pc.mu.RUnlock()
		return pc.handleUnknown(plugin, permission)
	}

	rule, ok := pluginRules[permission]
	if !ok {
		pc.mu.RUnlock()
		return pc.handleUnknown(plugin, permission)
	}

	if rule.granted {
		pc.mu.RUnlock()
		return PermissionAllowed
	}
	pc.mu.RUnlock()
	return PermissionDenied
}

// handleUnknown triggers consent flow for unknown permissions in enforce mode.
func (pc *PermissionChecker) handleUnknown(plugin, permission string) PermissionResult {
	pc.mu.RLock()
	mode := pc.mode
	prompt := pc.consentPrompt
	pc.mu.RUnlock()

	if mode == PermissionModeEnforce && prompt != nil {
		// Parse resource.action
		resource, action := splitPermission(permission)
		perms := []Permission{
			{Plugin: plugin, Resource: resource, Action: action},
		}
		prompt.Plugin = plugin
		prompt.Permissions = perms

		// Trigger consent flow
		response, err := prompt.Ask()
		if err == nil {
			switch response {
			case ConsentApprove, ConsentAlwaysApprove:
				_ = pc.Grant(plugin, permission)
				return PermissionAllowed
			case ConsentDeny, ConsentAlwaysDeny:
				_ = pc.Deny(plugin, permission)
				return PermissionDenied
			}
		}
	}

	return PermissionUnknown
}

// Grant grants a permission to a plugin. If the permission already exists,
// it is updated to granted. The granted_at timestamp is set to now.
func (pc *PermissionChecker) Grant(plugin, permission string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.rules[plugin] == nil {
		pc.rules[plugin] = make(map[string]*permissionRule)
	}

	pc.rules[plugin][permission] = &permissionRule{
		granted:   true,
		grantedAt: time.Now(),
	}
	return nil
}

// Deny denies a permission for a plugin. If the permission does not exist,
// it is created as denied.
func (pc *PermissionChecker) Deny(plugin, permission string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.rules[plugin] == nil {
		pc.rules[plugin] = make(map[string]*permissionRule)
	}

	pc.rules[plugin][permission] = &permissionRule{
		granted:   false,
		grantedAt: time.Now(),
	}
	return nil
}

// Revoke removes a permission rule for a plugin. After revocation,
// the permission returns PermissionUnknown.
func (pc *PermissionChecker) Revoke(plugin, permission string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pluginRules, ok := pc.rules[plugin]
	if !ok {
		return nil
	}

	delete(pluginRules, permission)
	if len(pluginRules) == 0 {
		delete(pc.rules, plugin)
	}
	return nil
}

// List returns all permissions for a given plugin.
func (pc *PermissionChecker) List(plugin string) []Permission {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	pluginRules, ok := pc.rules[plugin]
	if !ok {
		return nil
	}

	perms := make([]Permission, 0, len(pluginRules))
	for permStr, rule := range pluginRules {
		resource, action := splitPermission(permStr)
		perms = append(perms, Permission{
			Plugin:    plugin,
			Resource:  resource,
			Action:    action,
			Granted:   rule.granted,
			GrantedAt: rule.grantedAt,
		})
	}
	return perms
}

// Save persists the current permissions state to the configured file.
func (pc *PermissionChecker) Save() error {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if pc.filePath == "" {
		return fmt.Errorf("no file path configured for permissions")
	}

	// Build the YAML structure
	type permEntry struct {
		Plugin    string `yaml:"plugin"`
		Resource  string `yaml:"resource"`
		Action    string `yaml:"action"`
		Granted   bool   `yaml:"granted"`
		GrantedAt string `yaml:"granted_at"`
	}

	type config struct {
		Mode        string      `yaml:"mode"`
		Permissions []permEntry `yaml:"permissions"`
	}

	c := config{
		Mode: string(pc.mode),
	}

	for plugin, pluginRules := range pc.rules {
		for permStr, rule := range pluginRules {
			resource, action := splitPermission(permStr)
			c.Permissions = append(c.Permissions, permEntry{
				Plugin:    plugin,
				Resource:  resource,
				Action:    action,
				Granted:   rule.granted,
				GrantedAt: rule.grantedAt.Format(time.RFC3339),
			})
		}
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}

	if err := os.WriteFile(pc.filePath, data, 0644); err != nil {
		return fmt.Errorf("write permissions file: %w", err)
	}

	return nil
}

// Load reads permissions from the configured file. If the file does not exist,
// the checker remains in its current state.
func (pc *PermissionChecker) Load() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.filePath == "" {
		return nil
	}

	data, err := os.ReadFile(pc.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read permissions file: %w", err)
	}

	type permEntry struct {
		Plugin    string `yaml:"plugin"`
		Resource  string `yaml:"resource"`
		Action    string `yaml:"action"`
		Granted   bool   `yaml:"granted"`
		GrantedAt string `yaml:"granted_at"`
	}

	type config struct {
		Mode        string      `yaml:"mode"`
		Permissions []permEntry `yaml:"permissions"`
	}

	var c config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("parse permissions file: %w", err)
	}

	// Set mode
	if c.Mode != "" {
		pc.mode = PermissionMode(c.Mode)
	}

	// Reset rules
	pc.rules = make(map[string]map[string]*permissionRule)

	// Load permissions
	for _, pe := range c.Permissions {
		if pc.rules[pe.Plugin] == nil {
			pc.rules[pe.Plugin] = make(map[string]*permissionRule)
		}

		var grantedAt time.Time
		if pe.GrantedAt != "" {
			grantedAt, _ = time.Parse(time.RFC3339, pe.GrantedAt)
		}

		permKey := joinPermission(pe.Resource, pe.Action)
		pc.rules[pe.Plugin][permKey] = &permissionRule{
			granted:   pe.Granted,
			grantedAt: grantedAt,
		}
	}

	return nil
}

// splitPermission splits a "resource.action" permission string into its components.
func splitPermission(perm string) (resource, action string) {
	parts := strings.SplitN(perm, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return perm, ""
}

// joinPermission joins resource and action into a "resource.action" permission string.
func joinPermission(resource, action string) string {
	if action == "" {
		return resource
	}
	return resource + "." + action
}
