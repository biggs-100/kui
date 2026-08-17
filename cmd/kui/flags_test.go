package main

import (
	"testing"
)

func TestOptionsZeroValues(t *testing.T) {
	opts, remaining, err := parseFlags([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("remaining args = %v, want empty", remaining)
	}
	if opts.Model != "" {
		t.Errorf("Model = %q, want empty", opts.Model)
	}
	if opts.Tools != "" {
		t.Errorf("Tools = %q, want empty", opts.Tools)
	}
	if opts.ExcludeTools != "" {
		t.Errorf("ExcludeTools = %q, want empty", opts.ExcludeTools)
	}
	if opts.NoTools != false {
		t.Error("NoTools should be false")
	}
	if opts.NoExtensions != false {
		t.Error("NoExtensions should be false")
	}
	if opts.NoSkills != false {
		t.Error("NoSkills should be false")
	}
	if opts.NoSession != false {
		t.Error("NoSession should be false")
	}
	if opts.Verbose != false {
		t.Error("Verbose should be false")
	}
	if opts.Mode != "" {
		t.Errorf("Mode = %q, want empty", opts.Mode)
	}
	if opts.Approve != false {
		t.Error("Approve should be false")
	}
	if opts.Print != false {
		t.Error("Print should be false")
	}
	if opts.Thinking != "" {
		t.Errorf("Thinking = %q, want empty", opts.Thinking)
	}
}

// ---------------------------------------------------------------------------
// Phase 2: Core Parser
// ---------------------------------------------------------------------------

func TestParseFlagsLongFlagSpace(t *testing.T) {
	opts, remaining, err := parseFlags([]string{"--model", "gpt-4o"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", opts.Model, "gpt-4o")
	}
	if len(remaining) != 0 {
		t.Errorf("remaining = %v, want empty", remaining)
	}
}

func TestParseFlagsLongFlagEquals(t *testing.T) {
	opts, remaining, err := parseFlags([]string{"--model=gpt-4o"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", opts.Model, "gpt-4o")
	}
	if len(remaining) != 0 {
		t.Errorf("remaining = %v, want empty", remaining)
	}
}

func TestParseFlagsBoolFlag(t *testing.T) {
	opts, remaining, err := parseFlags([]string{"--verbose"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
	if len(remaining) != 0 {
		t.Errorf("remaining = %v, want empty", remaining)
	}
}

func TestParseFlagsShortFlag(t *testing.T) {
	opts, remaining, err := parseFlags([]string{"-m", "gpt-4o"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", opts.Model, "gpt-4o")
	}
	if len(remaining) != 0 {
		t.Errorf("remaining = %v, want empty", remaining)
	}
}

func TestParseFlagsSeparator(t *testing.T) {
	opts, remaining, err := parseFlags([]string{"--model", "gpt-4o", "--", "--verbose"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", opts.Model, "gpt-4o")
	}
	if len(remaining) != 1 || remaining[0] != "--verbose" {
		t.Errorf("remaining = %v, want [--verbose]", remaining)
	}
}

func TestParseFlagsRemainingPositional(t *testing.T) {
	opts, remaining, err := parseFlags([]string{"--model", "gpt-4o", "hello", "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", opts.Model, "gpt-4o")
	}
	if len(remaining) != 2 || remaining[0] != "hello" || remaining[1] != "world" {
		t.Errorf("remaining = %v, want [hello world]", remaining)
	}
}

// ---------------------------------------------------------------------------
// Phase 3: Error Handling
// ---------------------------------------------------------------------------

func TestParseFlagsUnknownFlag(t *testing.T) {
	_, _, err := parseFlags([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
	if !contains(err.Error(), "unknown-flag") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "unknown-flag")
	}
}

func TestParseFlagsUnknownShortFlag(t *testing.T) {
	_, _, err := parseFlags([]string{"-z"})
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
	if !contains(err.Error(), "-z") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "-z")
	}
}

func TestParseFlagsMissingValue(t *testing.T) {
	_, _, err := parseFlags([]string{"--model"})
	if err == nil {
		t.Fatal("expected error for missing value, got nil")
	}
	if !contains(err.Error(), "missing") && !contains(err.Error(), "value") {
		t.Errorf("error = %q, want it to indicate a missing value", err.Error())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Edge cases and additional flag coverage
// ---------------------------------------------------------------------------

func TestParseFlagsVerboseEqualsTrue(t *testing.T) {
	opts, _, err := parseFlags([]string{"--verbose=true"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
}

func TestParseFlagsVerboseEqualsFalse(t *testing.T) {
	opts, _, err := parseFlags([]string{"--verbose=false"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// --verbose=false is treated as a boolean flag (the value is ignored).
	if !opts.Verbose {
		t.Error("Verbose should be true even with =false (bool flag)")
	}
}

func TestParseFlagsVerboseSpaceTrue(t *testing.T) {
	opts, remaining, err := parseFlags([]string{"--verbose", "true"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
	// "true" is not consumed — it's a positional arg after a bool flag.
	if len(remaining) != 1 || remaining[0] != "true" {
		t.Errorf("remaining = %v, want [true]", remaining)
	}
}

func TestParseFlagsModeJson(t *testing.T) {
	opts, _, err := parseFlags([]string{"--mode", "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Mode != "json" {
		t.Errorf("Mode = %q, want %q", opts.Mode, "json")
	}
}

func TestParseFlagsModeText(t *testing.T) {
	opts, _, err := parseFlags([]string{"--mode", "text"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Mode != "text" {
		t.Errorf("Mode = %q, want %q", opts.Mode, "text")
	}
}

func TestParseFlagsModeEquals(t *testing.T) {
	opts, _, err := parseFlags([]string{"--mode=json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Mode != "json" {
		t.Errorf("Mode = %q, want %q", opts.Mode, "json")
	}
}

func TestParseFlagsApproveLong(t *testing.T) {
	opts, _, err := parseFlags([]string{"--approve"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Approve {
		t.Error("Approve should be true")
	}
}

func TestParseFlagsApproveShort(t *testing.T) {
	opts, _, err := parseFlags([]string{"-a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Approve {
		t.Error("Approve should be true")
	}
}

func TestParseFlagsPrintLong(t *testing.T) {
	opts, _, err := parseFlags([]string{"--print"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Print {
		t.Error("Print should be true")
	}
}

func TestParseFlagsPrintShort(t *testing.T) {
	opts, _, err := parseFlags([]string{"-p"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Print {
		t.Error("Print should be true")
	}
}

func TestParseFlagsNoExtensionsLong(t *testing.T) {
	opts, _, err := parseFlags([]string{"--no-extensions"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.NoExtensions {
		t.Error("NoExtensions should be true")
	}
}

func TestParseFlagsNoExtensionsShort(t *testing.T) {
	opts, _, err := parseFlags([]string{"-ne"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.NoExtensions {
		t.Error("NoExtensions should be true")
	}
}

func TestParseFlagsNoSkillsLong(t *testing.T) {
	opts, _, err := parseFlags([]string{"--no-skills"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.NoSkills {
		t.Error("NoSkills should be true")
	}
}

func TestParseFlagsNoSkillsShort(t *testing.T) {
	opts, _, err := parseFlags([]string{"-ns"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.NoSkills {
		t.Error("NoSkills should be true")
	}
}

func TestParseFlagsNoSession(t *testing.T) {
	opts, _, err := parseFlags([]string{"--no-session"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.NoSession {
		t.Error("NoSession should be true")
	}
}

func TestParseFlagsNoTools(t *testing.T) {
	opts, _, err := parseFlags([]string{"--no-tools"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.NoTools {
		t.Error("NoTools should be true")
	}
}

func TestParseFlagsToolsComma(t *testing.T) {
	opts, _, err := parseFlags([]string{"--tools", "bash,read_file"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Tools != "bash,read_file" {
		t.Errorf("Tools = %q, want %q", opts.Tools, "bash,read_file")
	}
}

func TestParseFlagsExcludeToolsComma(t *testing.T) {
	opts, _, err := parseFlags([]string{"--exclude-tools", "bash"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.ExcludeTools != "bash" {
		t.Errorf("ExcludeTools = %q, want %q", opts.ExcludeTools, "bash")
	}
}

func TestParseFlagsThinkingSpace(t *testing.T) {
	opts, _, err := parseFlags([]string{"--thinking", "high"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Thinking != "high" {
		t.Errorf("Thinking = %q, want %q", opts.Thinking, "high")
	}
}

func TestParseFlagsThinkingEquals(t *testing.T) {
	opts, _, err := parseFlags([]string{"--thinking=medium"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Thinking != "medium" {
		t.Errorf("Thinking = %q, want %q", opts.Thinking, "medium")
	}
}

func TestResolveThinkingInvalid(t *testing.T) {
	_, err := resolveThinking("banana")
	if err == nil {
		t.Fatal("expected error for invalid thinking level, got nil")
	}
	if !contains(err.Error(), "banana") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "banana")
	}
}

func TestResolveThinkingEmpty(t *testing.T) {
	level, err := resolveThinking("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if level != "off" {
		t.Errorf("level = %q, want %q", level, "off")
	}
}

func TestResolveThinkingValid(t *testing.T) {
	for _, want := range []string{"off", "low", "medium", "high"} {
		level, err := resolveThinking(want)
		if err != nil {
			t.Fatalf("resolveThinking(%q) error = %v", want, err)
		}
		if level != want {
			t.Errorf("resolveThinking(%q) = %q, want %q", want, level, want)
		}
	}
}

func TestParseFlagsMultipleFlags(t *testing.T) {
	opts, remaining, err := parseFlags([]string{"--model", "gpt-4o", "--verbose", "-a", "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", opts.Model, "gpt-4o")
	}
	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
	if !opts.Approve {
		t.Error("Approve should be true")
	}
	if len(remaining) != 1 || remaining[0] != "hello" {
		t.Errorf("remaining = %v, want [hello]", remaining)
	}
}

func TestParseFlagsAllTogether(t *testing.T) {
	args := []string{
		"--model", "gpt-4o",
		"--tools", "bash,read_file",
		"--exclude-tools", "bash",
		"--no-extensions",
		"--no-skills",
		"--mode", "json",
		"--verbose",
		"--approve",
		"--print",
		"--no-session",
		"hello world",
	}
	opts, remaining, err := parseFlags(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Model != "gpt-4o" {
		t.Errorf("Model = %q", opts.Model)
	}
	if opts.Tools != "bash,read_file" {
		t.Errorf("Tools = %q", opts.Tools)
	}
	if opts.ExcludeTools != "bash" {
		t.Errorf("ExcludeTools = %q", opts.ExcludeTools)
	}
	if !opts.NoExtensions {
		t.Error("NoExtensions should be true")
	}
	if !opts.NoSkills {
		t.Error("NoSkills should be true")
	}
	if opts.Mode != "json" {
		t.Errorf("Mode = %q", opts.Mode)
	}
	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
	if !opts.Approve {
		t.Error("Approve should be true")
	}
	if !opts.Print {
		t.Error("Print should be true")
	}
	if !opts.NoSession {
		t.Error("NoSession should be true")
	}
	if len(remaining) != 1 || remaining[0] != "hello world" {
		t.Errorf("remaining = %v, want [hello world]", remaining)
	}
}