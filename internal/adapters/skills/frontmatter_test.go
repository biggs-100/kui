package skills

import (
	"testing"
)

func TestParseFrontmatterValid(t *testing.T) {
	// REQ-RS-3, REQ-RS-20: ParseFrontmatter extracts name, description,
	// triggers from a valid --- delimited YAML header.
	content := []byte(`---
name: go-testing
description: Run and debug Go tests
triggers:
  - go test
  - testing
---

# Body content here
`)
	meta, body, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("ParseFrontmatter returned error: %v", err)
	}
	if meta.Name != "go-testing" {
		t.Errorf("Name = %q, want %q", meta.Name, "go-testing")
	}
	if meta.Description != "Run and debug Go tests" {
		t.Errorf("Description = %q, want %q", meta.Description, "Run and debug Go tests")
	}
	if len(meta.Triggers) != 2 || meta.Triggers[0] != "go test" || meta.Triggers[1] != "testing" {
		t.Errorf("Triggers = %v, want [go test, testing]", meta.Triggers)
	}
	wantBody := "# Body content here\n"
	if body != wantBody {
		t.Errorf("body = %q, want %q", body, wantBody)
	}
}

func TestParseFrontmatterMissingFields(t *testing.T) {
	// REQ-RS-3: ParseFrontmatter handles missing fields gracefully.
	content := []byte(`---
name: simple-skill
---

# Hello
`)
	meta, body, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("ParseFrontmatter returned error: %v", err)
	}
	if meta.Name != "simple-skill" {
		t.Errorf("Name = %q, want %q", meta.Name, "simple-skill")
	}
	if meta.Description != "" {
		t.Errorf("Description = %q, want empty", meta.Description)
	}
	if len(meta.Triggers) != 0 {
		t.Errorf("Triggers = %v, want empty", meta.Triggers)
	}
	if body != "# Hello\n" {
		t.Errorf("body = %q, want %q", body, "# Hello\n")
	}
}

func TestParseFrontmatterNoFrontmatter(t *testing.T) {
	// REQ-RS-3: A file with no frontmatter returns empty meta, no error.
	content := []byte(`# Just a heading

Some body content.
`)
	meta, body, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("ParseFrontmatter returned error: %v", err)
	}
	if meta.Name != "" {
		t.Errorf("Name = %q, want empty", meta.Name)
	}
	if meta.Description != "" {
		t.Errorf("Description = %q, want empty", meta.Description)
	}
	if body != "# Just a heading\n\nSome body content.\n" {
		t.Errorf("body = %q, want full content", body)
	}
}

func TestParseFrontmatterEmptyFile(t *testing.T) {
	// An empty file returns empty meta and empty body, no error.
	meta, body, err := ParseFrontmatter([]byte{})
	if err != nil {
		t.Fatalf("ParseFrontmatter returned error: %v", err)
	}
	if meta.Name != "" {
		t.Errorf("Name = %q, want empty", meta.Name)
	}
	if body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

func TestParseFrontmatterMalformedYAML(t *testing.T) {
	// Malformed YAML in frontmatter returns an error.
	content := []byte(`---
name: broken
triggers: [invalid yaml
---
`)
	_, _, err := ParseFrontmatter(content)
	if err == nil {
		t.Fatal("ParseFrontmatter on malformed YAML should return error, got nil")
	}
}
