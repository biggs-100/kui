// Package permissions implements the ordered allow/ask/deny ruleset that
// gates tool advertisement and dispatch (D15, REQ-PERM-1..4). It parses no
// files: rule layers are constructed by callers (the profile loader, PR 3)
// and merged here with last-rule-wins semantics.
package permissions

import (
	"path"

	"github.com/biggs-100/kui/internal/core"
)

// Decision is the verdict a rule attaches to a tool name.
type Decision string

// The three rule actions (REQ-PERM-1). Ask is reserved for future interactive
// prompts; it currently degrades to Deny (REQ-PERM-2).
const (
	Allow Decision = "allow"
	Ask   Decision = "ask"
	Deny  Decision = "deny"
)

// Rule binds a tool-name pattern (a plain name or a path.Match glob such as
// "*" or "read_*") to a decision. Patterns evaluate against tool names only,
// so a rule naming an unregistered tool is simply irrelevant (REQ-PERM-1).
type Rule struct {
	Pattern string
	Action  Decision
}

// Ruleset is an ordered rule list evaluated last-match-wins (REQ-PERM-1). It
// implements core.PermissionEvaluator.
type Ruleset struct {
	rules []Rule
}

// Flatten merges rule layers (defaults → config → profile) into one ordered
// ruleset; rules from later layers come after earlier ones, so the last
// matching rule wins (D15). An empty ruleset allows every tool.
func Flatten(layers ...[]Rule) *Ruleset {
	rs := &Ruleset{}
	for _, layer := range layers {
		rs.rules = append(rs.rules, layer...)
	}
	return rs
}

// NewPermissive returns a ruleset that allows every tool. Used by --approve
// to bypass all permission checks (REQ-CLI-26).
func NewPermissive() *Ruleset {
	return &Ruleset{}
}

// Evaluate resolves the decision for a tool name: the last matching rule
// wins, ask rules degrade to deny as the safe interim behavior (REQ-PERM-2),
// and a tool matching no rule is allowed (REQ-PERM-1).
func (r *Ruleset) Evaluate(name string) Decision {
	decision := Allow
	for _, rule := range r.rules {
		if match(rule.Pattern, name) {
			if rule.Action == Ask {
				decision = Deny
			} else {
				decision = rule.Action
			}
		}
	}
	return decision
}

// Allow reports whether the tool may be advertised and dispatched
// (core.PermissionEvaluator).
func (r *Ruleset) Allow(name string) bool {
	return r.Evaluate(name) == Allow
}

// Filter drops denied tools from the advertised slice, preserving order
// (core.PermissionEvaluator, D15, REQ-PERM-3).
func (r *Ruleset) Filter(tools []core.Tool) []core.Tool {
	filtered := make([]core.Tool, 0, len(tools))
	for _, tool := range tools {
		if r.Allow(tool.Name()) {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// match applies the rule pattern to the tool name. A malformed pattern never
// matches, so a broken rule is ignored without error (REQ-PERM-1).
func match(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return err == nil && ok
}
