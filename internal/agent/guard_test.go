package agent

import (
	"os/exec"
	"strings"
	"testing"
)

// TestAgentImportsNoIOOrYaml guards the adapter boundary (REQ-PROFILE-5,
// D19/D21): the agent wrapper wires ports only — all yaml and filesystem work
// lives in the adapters. It inspects the package's direct (non-test) imports
// via `go list` and fails on any banned package. Test-only imports are not
// considered, since tests may construct fixtures.
func TestAgentImportsNoIOOrYaml(t *testing.T) {
	out, err := exec.Command("go", "list", "-f", `{{join .Imports "\n"}}`, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list failed: %v\n%s", err, out)
	}
	banned := map[string]string{
		"os":               "filesystem access",
		"os/exec":          "subprocess execution",
		"io/ioutil":        "filesystem access",
		"io/fs":            "filesystem access",
		"path/filepath":    "filesystem path handling",
		"gopkg.in/yaml.v3": "yaml parsing",
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		if reason, ok := banned[line]; ok {
			t.Errorf("agent imports %q (%s) — IO belongs in adapters", line, reason)
		}
	}
}
