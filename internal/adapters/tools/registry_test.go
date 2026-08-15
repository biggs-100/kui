package tools

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestDefaultSetEnumeratesBuiltins covers REQ-TOOLS-4 "Enumerate built-in
// tools": the default tool set exposes read_file, write_file, and bash in
// stable advertisement order, each with a name, a description, and a valid
// JSON schema.
func TestDefaultSetEnumeratesBuiltins(t *testing.T) {
	set := Default(t.TempDir(), 0)

	if len(set) != 3 {
		t.Fatalf("default set has %d tools, want 3", len(set))
	}
	var names []string
	for _, tool := range set {
		names = append(names, tool.Name())
		if tool.Description() == "" {
			t.Errorf("tool %q has an empty description", tool.Name())
		}
		schema := tool.Schema()
		if schema == "" || !json.Valid([]byte(schema)) {
			t.Errorf("tool %q has an invalid schema %q", tool.Name(), schema)
		}
	}
	want := []string{"read_file", "write_file", "bash"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("default set order = %v, want %v", names, want)
	}
}
