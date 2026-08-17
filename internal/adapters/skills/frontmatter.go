package skills

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Frontmatter holds the optional YAML header from a SKILL.md file.
type Frontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Triggers    []string `yaml:"triggers"`
}

// ParseFrontmatter extracts YAML frontmatter from a --- delimited header.
// Returns the parsed metadata, the body content after the closing ---, and
// any error. If no frontmatter is present, returns empty metadata and the
// full content as body.
func ParseFrontmatter(data []byte) (*Frontmatter, string, error) {
	// Look for opening --- at the very start (after optional whitespace).
	trimmed := bytes.TrimLeft(data, " \t")
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return &Frontmatter{}, string(data), nil
	}

	// Find the closing ---
	rest := bytes.TrimPrefix(trimmed, []byte("---"))
	rest = bytes.TrimLeft(rest, "\n\r")

	idx := bytes.Index(rest, []byte("\n---"))
	if idx == -1 {
		// Also try --- at the very start of rest (empty frontmatter).
		idx = bytes.Index(rest, []byte("---"))
		if idx != 0 {
			return nil, "", fmt.Errorf("unterminated frontmatter: no closing ---")
		}
		// Empty frontmatter: ---\n---.
		return &Frontmatter{}, string(rest[3:]), nil
	}

	yamlBlock := rest[:idx]
	// Body starts after the closing \n--- line (4 bytes: \n---).
	bodyStart := idx + 4
	// Trim leading newlines so body starts cleanly.
	for bodyStart < len(rest) && rest[bodyStart] == '\n' {
		bodyStart++
	}
	body := string(rest[bodyStart:])

	var meta Frontmatter
	if err := yaml.Unmarshal(yamlBlock, &meta); err != nil {
		return nil, "", fmt.Errorf("parsing frontmatter: %w", err)
	}
	return &meta, body, nil
}
