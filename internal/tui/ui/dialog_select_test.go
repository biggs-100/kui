package ui_test

import (
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/tui/theme"
	"github.com/biggs-100/kui/internal/tui/ui"
)

func testDialogStyles() *theme.Styles {
	return theme.NewStyles(theme.DefaultTheme())
}

// TestWeightedFuzzySort verifies title*2+category weighting: titles rank above category-only matches.
func TestDialogSelectWeightedFuzzySort(t *testing.T) {
	items := []ui.SelectItem[string]{
		{Title: "model-a", Category: "AI", Value: "a"},
		{Title: "foo", Category: "model-stuff", Value: "b"},
		{Title: "model-b", Category: "AI", Value: "c"},
	}
	ds := ui.NewDialogSelect(items, 88, 10)
	ds.SetFlat(true)
	ds.Filter("mod")
	filtered := ds.Filtered()
	if len(filtered) == 0 {
		t.Fatal("expected filtered results for 'mod'")
	}
	// First result should be title match (model-a or model-b), not category-only foo
	if filtered[0].Title == "foo" {
		t.Errorf("weighted sort failed: first result title=%q, want model title first, got %#v", filtered[0].Title, filtered)
	}
	// Both title matches should be before category-only
	foundFoo := -1
	for i, it := range filtered {
		if it.Title == "foo" {
			foundFoo = i
			break
		}
	}
	if foundFoo != -1 && foundFoo < 2 {
		t.Errorf("category-only 'foo' ranked too high at %d, expected after title matches: %#v", foundFoo, filtered)
	}
}

func TestDialogSelectGrouping(t *testing.T) {
	items := []ui.SelectItem[string]{
		{Title: "cmd1", Category: "Session", Value: "1"},
		{Title: "cmd2", Category: "Edit", Value: "2"},
		{Title: "cmd3", Category: "Session", Value: "3"},
	}
	ds := ui.NewDialogSelect(items, 88, 10)
	ds.SetStyles(testDialogStyles())
	ds.SetFlat(false)
	view := ds.View(120, 30)
	if !strings.Contains(view, "Session") {
		t.Errorf("grouping should show Category header 'Session', got:\n%s", view)
	}
	if !strings.Contains(view, "Edit") {
		t.Errorf("grouping should show Category header 'Edit', got:\n%s", view)
	}
}

func TestDialogSelectSelectedUsesBackgroundMenu(t *testing.T) {
	items := []ui.SelectItem[string]{
		{Title: "a", Category: "X", Detail: strings.Repeat("detail ", 20), Value: "a"},
		{Title: "b", Category: "X", Detail: strings.Repeat("detail ", 20), Value: "b"},
		{Title: "c", Category: "X", Detail: strings.Repeat("detail ", 20), Value: "c"},
	}
	ds := ui.NewDialogSelect(items, 88, 10)
	ds.SetStyles(testDialogStyles())
	ds.SetFlat(true)
	ds.SetSelected(2)
	view := ds.View(120, 30)
	// Text dump should show > marker + truncated detail at 76 cols visible
	if !strings.Contains(view, ">") {
		t.Errorf("selected row should have > marker, got:\n%s", view)
	}
	// Detail should be truncated to 76 via ... middle
	if !strings.Contains(view, "...") {
		t.Errorf("detail should be truncated with ..., got:\n%s", view)
	}
}

func TestDialogSelectTruncateMiddle76(t *testing.T) {
	// Direct test of truncate middle logic via view
	longDetail := strings.Repeat("x", 100)
	items := []ui.SelectItem[string]{
		{Title: "t", Category: "C", Detail: longDetail, Value: "v"},
	}
	ds := ui.NewDialogSelect(items, 88, 10)
	ds.SetStyles(testDialogStyles())
	ds.SetFlat(true)
	ds.SetSelected(0)
	view := ds.View(120, 30)
	// Detail truncated to 76 cols should contain ...
	if !strings.Contains(view, "...") {
		t.Errorf("detail 100 chars should be truncated76 with ..., got view len %d", len(view))
	}
}

func TestDialogSelectEmptyView(t *testing.T) {
	ds := ui.NewDialogSelect[string](nil, 88, 10)
	ds.SetStyles(testDialogStyles())
	ds.SetEmptyView("no results")
	view := ds.View(120, 30)
	if !strings.Contains(view, "no results") {
		t.Errorf("empty view should show 'no results', got:\n%s", view)
	}
}

func TestDialogSelectPreserveSelection(t *testing.T) {
	items := []ui.SelectItem[string]{
		{Title: "apple", Category: "Fruit", Value: "apple"},
		{Title: "banana", Category: "Fruit", Value: "banana"},
		{Title: "cherry", Category: "Fruit", Value: "cherry"},
	}
	ds := ui.NewDialogSelect(items, 88, 10)
	ds.SetFlat(true)
	ds.SetSelected(1) // banana
	// Filter to keep banana
	ds.Filter("ban")
	if ds.SelectedItem() == nil || ds.SelectedItem().Value != "banana" {
		t.Errorf("preserveSelection should keep banana after filter, got %#v", ds.SelectedItem())
	}
	// Filter to exclude banana, selection should move to first available
	ds.Filter("cher")
	if ds.SelectedItem() == nil || ds.SelectedItem().Value != "cherry" {
		t.Errorf("after filter excludes previous, selection should be cherry, got %#v", ds.SelectedItem())
	}
}

func TestDialogSelectEscFilterThenClose(t *testing.T) {
	items := []ui.SelectItem[string]{
		{Title: "a", Category: "X", Value: "a"},
		{Title: "b", Category: "X", Value: "b"},
	}
	ds := ui.NewDialogSelect(items, 88, 10)
	ds.Filter("a")
	if ds.FilterText() != "a" {
		t.Fatalf("filter should be 'a', got %q", ds.FilterText())
	}
	// First Esc should clear filter, not close
	closed := ds.HandleEsc()
	if closed {
		t.Error("first Esc with filter non-empty should NOT close, should clear filter")
	}
	if ds.FilterText() != "" {
		t.Errorf("after first Esc, filter should be cleared, got %q", ds.FilterText())
	}
	if ds.IsQuitting() {
		t.Error("should not be quitting after first Esc")
	}
	// Second Esc should close
	closed = ds.HandleEsc()
	if !closed {
		t.Error("second Esc with empty filter should close")
	}
	if !ds.IsQuitting() {
		t.Error("should be quitting after second Esc")
	}
}

func TestDialogSelectDisabledSkip(t *testing.T) {
	items := []ui.SelectItem[string]{
		{Title: "enabled", Category: "X", Value: "e", Disabled: false},
		{Title: "opencode/gpt-nano", Category: "X", Value: "nano", Disabled: true},
		{Title: "enabled2", Category: "X", Value: "e2", Disabled: false},
	}
	ds := ui.NewDialogSelect(items, 88, 10)
	ds.SetFlat(true)
	ds.SetSelected(0)
	ds.MoveDown()
	// Should skip disabled nano and land on enabled2
	if ds.SelectedItem() == nil || ds.SelectedItem().Value != "e2" {
		t.Errorf("MoveDown should skip disabled nano, got %#v", ds.SelectedItem())
	}
}
