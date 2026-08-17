package views

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func TestHeaderTwoTabsActive(t *testing.T) {
	m := NewHeaderModel([]string{"coder", "writer"}, 0)
	got := m.Render()

	golden := filepath.Join("testdata", "header_two_tabs_active.txt")
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden file not found (run with -update): %v", err)
	}

	if got != string(want) {
		t.Errorf("header mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestHeaderNoProfiles(t *testing.T) {
	m := NewHeaderModel(nil, 0)
	got := m.Render()

	golden := filepath.Join("testdata", "header_no_profiles.txt")
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden file not found (run with -update): %v", err)
	}

	if got != string(want) {
		t.Errorf("header mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestHeaderActiveMarkedSecond(t *testing.T) {
	m := NewHeaderModel([]string{"coder", "writer"}, 1)
	got := m.Render()

	golden := filepath.Join("testdata", "header_second_active.txt")
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden file not found (run with -update): %v", err)
	}

	if got != string(want) {
		t.Errorf("header mismatch\ngot:\n%s\nwant:\n%s", got, string(want))
	}
}

func TestHeaderContainsProfileNames(t *testing.T) {
	m := NewHeaderModel([]string{"coder", "writer"}, 0)
	got := m.Render()
	if !strings.Contains(got, "coder") {
		t.Error("header should contain 'coder'")
	}
	if !strings.Contains(got, "writer") {
		t.Error("header should contain 'writer'")
	}
}

func TestHeaderNoProfilesContainsHint(t *testing.T) {
	m := NewHeaderModel(nil, 0)
	got := m.Render()
	// The hint text should be present when no profiles are provided
	if strings.TrimSpace(got) == "" {
		t.Error("header should render a hint when no profiles, got empty string")
	}
}
