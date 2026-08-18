package orchestration

import (
	"strings"
	"testing"
)

// ─── Task 3.10: RED — Test aggregator merge mode ───

func TestAggregateMerge(t *testing.T) {
	agg := &ResultAggregator{}

	results := []*SpawnResult{
		{Output: "agent A output"},
		{Output: "agent B output"},
		{Output: "agent C output"},
	}

	got := agg.Aggregate(results, "merge")

	if !strings.Contains(got, "agent A output") {
		t.Error("merge should contain first output")
	}
	if !strings.Contains(got, "agent B output") {
		t.Error("merge should contain second output")
	}
	if !strings.Contains(got, "agent C output") {
		t.Error("merge should contain third output")
	}
	// Verify separator exists between outputs
	if !strings.Contains(got, "---") {
		t.Error("merge should use separator between outputs")
	}
}

// ─── Task 3.10: RED — Test aggregator summary mode ───

func TestAggregateSummary(t *testing.T) {
	agg := &ResultAggregator{}

	results := []*SpawnResult{
		{Output: "finding 1"},
		{Output: "finding 2"},
	}

	got := agg.Aggregate(results, "summary")

	// Summary joins outputs (may include prefix/labeling)
	if !strings.Contains(got, "finding 1") {
		t.Error("summary should contain first output")
	}
	if !strings.Contains(got, "finding 2") {
		t.Error("summary should contain second output")
	}
}

// ─── Task 3.10: RED — Test aggregator select mode ───

func TestAggregateSelect(t *testing.T) {
	agg := &ResultAggregator{}

	results := []*SpawnResult{
		{Output: "short"},
		{Output: "this is the longest output with the most detail"},
		{Output: "medium length"},
	}

	got := agg.Aggregate(results, "select")

	// Select picks the longest output
	if got != "this is the longest output with the most detail" {
		t.Errorf("select should pick longest output, got: %q", got)
	}
}

// ─── Task 3.10: RED — Test aggregator default mode ───

func TestAggregateDefault(t *testing.T) {
	agg := &ResultAggregator{}

	results := []*SpawnResult{
		{Output: "output A"},
		{Output: "output B"},
	}

	got := agg.Aggregate(results, "unknown_mode")

	// Unknown mode defaults to merge
	if !strings.Contains(got, "output A") {
		t.Error("default mode should contain first output")
	}
	if !strings.Contains(got, "output B") {
		t.Error("default mode should contain second output")
	}
}

// ─── Edge case: Empty results ───

func TestAggregateEmpty(t *testing.T) {
	agg := &ResultAggregator{}

	got := agg.Aggregate(nil, "merge")
	if got == "" {
		t.Error("aggregate empty results should return non-empty string (empty marker)")
	}
}

// ─── Edge case: Single result ───

func TestAggregateSingleResult(t *testing.T) {
	agg := &ResultAggregator{}

	results := []*SpawnResult{
		{Output: "only one"},
	}

	got := agg.Aggregate(results, "merge")
	if got != "only one" {
		t.Errorf("single result merge should return the output, got: %q", got)
	}
}

// ─── Edge case: Select with all same length ───

func TestAggregateSelectSameLength(t *testing.T) {
	agg := &ResultAggregator{}

	results := []*SpawnResult{
		{Output: "aaa"},
		{Output: "bbb"},
		{Output: "ccc"},
	}

	got := agg.Aggregate(results, "select")
	// All same length — should pick first one
	if got != "aaa" {
		t.Errorf("select with same length should pick first, got: %q", got)
	}
}

// ─── Edge case: Results with error ───

func TestAggregateWithErrors(t *testing.T) {
	agg := &ResultAggregator{}

	results := []*SpawnResult{
		{Output: "good output"},
		{Output: "", Error: assertError("failed")},
	}

	got := agg.Aggregate(results, "merge")
	if !strings.Contains(got, "good output") {
		t.Error("merge should include good output even when other results have errors")
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }
