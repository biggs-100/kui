package ui

import (
	"testing"
)

func TestSplitBorderChars(t *testing.T) {
	if SplitBorder.Left != "┃" {
		t.Errorf("SplitBorder.Left = %q, want %q", SplitBorder.Left, "┃")
	}
	if SplitBorder.Bottom != "╹" {
		t.Errorf("SplitBorder.Bottom = %q, want %q", SplitBorder.Bottom, "╹")
	}
	// Ensure not drifted to │ or └
	if SplitBorder.Left == "│" {
		t.Error("SplitBorder.Left drifted to │, must be ┃")
	}
	if SplitBorder.Bottom == "└" || SplitBorder.Bottom == "│" {
		t.Errorf("SplitBorder.Bottom drifted to %q, must be ╹", SplitBorder.Bottom)
	}
}

func TestEmptyBorder(t *testing.T) {
	// EmptyBorder should have no visible chars
	if EmptyBorder.Left != "" && EmptyBorder.Left != " " {
		t.Errorf("EmptyBorder.Left = %q, want empty", EmptyBorder.Left)
	}
	if EmptyBorder.Bottom != "" && EmptyBorder.Bottom != " " {
		t.Errorf("EmptyBorder.Bottom = %q, want empty", EmptyBorder.Bottom)
	}
}

func TestPromptDecorativeBottom(t *testing.T) {
	if PromptBottom != "▀" {
		t.Errorf("PromptBottom = %q, want %q", PromptBottom, "▀")
	}
}
