package core

import (
	"os/exec"
	"strings"
	"testing"
)

// TestCoreImportsStdlibOnly guards the hexagonal boundary (D1): the domain
// core must never depend on third-party packages. It inspects the package's
// real dependency graph via `go list -deps` and fails on any import outside
// the standard library.
func TestCoreImportsStdlibOnly(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list failed: %v\n%s", err, out)
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		if line == "github.com/biggs-100/kui/internal/core" {
			continue // the package itself is not a dependency
		}
		t.Errorf("core imports non-stdlib package %q", line)
	}
}
