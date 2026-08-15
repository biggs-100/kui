package permissions

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/biggs-100/kui/internal/core"
)

// fakeTool implements core.Tool so Filter tests exercise the real adapter
// boundary with the actual Tool interface.
type fakeTool struct{ name string }

func (f fakeTool) Name() string        { return f.name }
func (f fakeTool) Description() string { return "fake tool: " + f.name }
func (f fakeTool) Schema() string      { return `{"type":"object","properties":{}}` }
func (f fakeTool) Execute(context.Context, json.RawMessage) (string, error) {
	return "", nil
}

func toolNames(tools []core.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name())
	}
	return names
}

// TestEvaluateLastRuleWins covers REQ-PERM-1 "Last rule wins": the final
// matching rule decides, even when an earlier rule also matched.
func TestEvaluateLastRuleWins(t *testing.T) {
	rs := Flatten([]Rule{
		{Pattern: "read_*", Action: Allow},
		{Pattern: "read_secrets", Action: Deny},
	})

	if got := rs.Evaluate("read_secrets"); got != Deny {
		t.Errorf("Evaluate(read_secrets) = %q, want deny (last rule wins)", got)
	}
	if got := rs.Evaluate("read_notes"); got != Allow {
		t.Errorf("Evaluate(read_notes) = %q, want allow (only the first rule matches)", got)
	}
}

// TestEvaluateWildcardMatch covers REQ-PERM-1 "Wildcard match": glob patterns
// match by name, and a non-matching name falls through to allow.
func TestEvaluateWildcardMatch(t *testing.T) {
	rs := Flatten([]Rule{{Pattern: "bash_*", Action: Deny}})

	if got := rs.Evaluate("bash_run"); got != Deny {
		t.Errorf("Evaluate(bash_run) = %q, want deny", got)
	}
	if got := rs.Evaluate("bash"); got != Allow {
		t.Errorf("Evaluate(bash) = %q, want allow (no matching rule)", got)
	}
}

// TestEvaluateEmptyRulesetAllows covers REQ-PERM-1 "Empty ruleset": with no
// rules every tool is allowed.
func TestEvaluateEmptyRulesetAllows(t *testing.T) {
	if got := Flatten().Evaluate("read_file"); got != Allow {
		t.Errorf("Flatten() Evaluate(read_file) = %q, want allow", got)
	}
	if got := Flatten([]Rule{}).Evaluate("read_file"); got != Allow {
		t.Errorf("Flatten([]Rule{}) Evaluate(read_file) = %q, want allow", got)
	}
}

// TestEvaluateUnregisteredToolRuleIgnored covers REQ-PERM-1 "Rule for
// unregistered tool": a rule naming a tool that is not registered is ignored
// without error when other tools are evaluated; it still applies to its own
// name because rules evaluate against names, not the registry.
func TestEvaluateUnregisteredToolRuleIgnored(t *testing.T) {
	rs := Flatten([]Rule{{Pattern: "not_registered", Action: Deny}})

	if got := rs.Evaluate("read_file"); got != Allow {
		t.Errorf("Evaluate(read_file) = %q, want allow (rule for an unregistered tool must not affect it)", got)
	}
	if got := rs.Evaluate("not_registered"); got != Deny {
		t.Errorf("Evaluate(not_registered) = %q, want deny (the rule applies by name)", got)
	}
}

// TestEvaluateAskDegradesToDeny covers REQ-PERM-2: ask rules never prompt;
// they evaluate to deny as the safe interim behavior.
func TestEvaluateAskDegradesToDeny(t *testing.T) {
	rs := Flatten([]Rule{{Pattern: "write_*", Action: Ask}})

	if got := rs.Evaluate("write_file"); got != Deny {
		t.Errorf("Evaluate(write_file) = %q, want deny (ask degrades to deny)", got)
	}
}

// TestFlattenLayerOrder verifies the defaults → config → profile layering:
// later layers append after earlier ones, so their rules win on conflict.
func TestFlattenLayerOrder(t *testing.T) {
	rs := Flatten(
		[]Rule{{Pattern: "*", Action: Deny}},
		[]Rule{{Pattern: "read_file", Action: Allow}},
	)

	if got := rs.Evaluate("read_file"); got != Allow {
		t.Errorf("Evaluate(read_file) = %q, want allow (profile layer wins)", got)
	}
	if got := rs.Evaluate("write_file"); got != Deny {
		t.Errorf("Evaluate(write_file) = %q, want deny (default layer still applies)", got)
	}
}

// TestRulesetAllow drives the core.PermissionEvaluator entry point.
func TestRulesetAllow(t *testing.T) {
	rs := Flatten([]Rule{{Pattern: "bash", Action: Deny}})

	if rs.Allow("read_file") != true {
		t.Error("Allow(read_file) = false, want true")
	}
	if rs.Allow("bash") != false {
		t.Error("Allow(bash) = true, want false")
	}
}

// TestRulesetFilterDropsDeniedTools drives Filter over the core.Tool slice,
// proving denied tools are removed while allowed ones keep their order.
func TestRulesetFilterDropsDeniedTools(t *testing.T) {
	rs := Flatten([]Rule{
		{Pattern: "*", Action: Deny},
		{Pattern: "read_*", Action: Allow},
	})
	tools := []core.Tool{
		fakeTool{name: "read_file"},
		fakeTool{name: "bash"},
		fakeTool{name: "write_file"},
	}

	got := toolNames(rs.Filter(tools))
	if want := []string{"read_file"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Filter(tools) = %v, want %v", got, want)
	}
}

// TestEvaluateMalformedPatternIgnored triangulates the match error path: a
// broken glob never matches, so the rule is ignored without error.
func TestEvaluateMalformedPatternIgnored(t *testing.T) {
	rs := Flatten([]Rule{
		{Pattern: "[", Action: Deny},
		{Pattern: "*", Action: Allow},
	})

	if got := rs.Evaluate("read_file"); got != Allow {
		t.Errorf("Evaluate(read_file) = %q, want allow (malformed pattern must be ignored)", got)
	}
}
