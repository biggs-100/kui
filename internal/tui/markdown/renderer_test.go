package markdown

import (
	"strings"
	"testing"

	"github.com/biggs-100/kui/internal/tui/theme"
)

func testStyles() *theme.Styles {
	return theme.NewStyles(theme.DefaultTheme())
}

func testTheme() *theme.Theme {
	return theme.DefaultTheme()
}

func TestRenderHeading(t *testing.T) {
	got := Render("# Hello World", testStyles())
	if got == "" {
		t.Fatal("Render returned empty string")
	}
	if !strings.Contains(got, "Hello World") {
		t.Errorf("Render should contain heading text, got: %q", got)
	}
}

func TestRenderBold(t *testing.T) {
	got := Render("This is **bold** text", testStyles())
	if got == "" {
		t.Fatal("Render returned empty string")
	}
	if !strings.Contains(got, "bold") {
		t.Errorf("Render should contain bold text, got: %q", got)
	}
}

func TestRenderItalic(t *testing.T) {
	got := Render("This is *italic* text", testStyles())
	if got == "" {
		t.Fatal("Render returned empty string")
	}
	if !strings.Contains(got, "italic") {
		t.Errorf("Render should contain italic text, got: %q", got)
	}
}

func TestRenderInlineCode(t *testing.T) {
	got := Render("Use `fmt.Println()` to print", testStyles())
	if got == "" {
		t.Fatal("Render returned empty string")
	}
	if !strings.Contains(got, "fmt.Println()") {
		t.Errorf("Render should contain inline code, got: %q", got)
	}
}

func TestRenderList(t *testing.T) {
	input := "- item one\n- item two\n- item three"
	got := Render(input, testStyles())
	if got == "" {
		t.Fatal("Render returned empty string")
	}
	if !strings.Contains(got, "item one") {
		t.Errorf("Render should contain list items, got: %q", got)
	}
}

func TestRenderBlockquote(t *testing.T) {
	got := Render("> This is a quote", testStyles())
	if got == "" {
		t.Fatal("Render returned empty string")
	}
	if !strings.Contains(got, "This is a quote") {
		t.Errorf("Render should contain blockquote text, got: %q", got)
	}
}

func TestRenderFence(t *testing.T) {
	input := "```go\nfunc main() {}\n```"
	got := Render(input, testStyles())
	if got == "" {
		t.Fatal("Render returned empty string")
	}
	// Fenced code should be syntax-highlighted (contains ANSI codes)
	if !strings.Contains(got, "func") {
		t.Errorf("Render should contain fenced code, got: %q", got)
	}
}

func TestRenderPlainPassthrough(t *testing.T) {
	input := "just plain text with no markdown"
	got := Render(input, testStyles())
	if !strings.Contains(got, "just plain text with no markdown") {
		t.Errorf("plain text should pass through, got: %q", got)
	}
}

func TestRenderEmpty(t *testing.T) {
	got := Render("", testStyles())
	if got != "" {
		t.Errorf("empty input should return empty, got: %q", got)
	}
}

func TestRenderMixed(t *testing.T) {
	input := "# Title\n\nSome **bold** text with `code`.\n\n- list item\n- another"
	got := Render(input, testStyles())
	if got == "" {
		t.Fatal("Render returned empty string")
	}
	if !strings.Contains(got, "Title") {
		t.Error("should contain heading")
	}
	if !strings.Contains(got, "bold") {
		t.Error("should contain bold text")
	}
	if !strings.Contains(got, "code") {
		t.Error("should contain inline code")
	}
	if !strings.Contains(got, "list item") {
		t.Error("should contain list item")
	}
}

func TestHighlightCode(t *testing.T) {
	code := `package main

import "fmt"

func main() {
    fmt.Println("hello")
}`
	got := HighlightCode(code, "go", testTheme())
	if got == "" {
		t.Fatal("HighlightCode returned empty string")
	}
	if !strings.Contains(got, "func") {
		t.Error("highlighted output should contain 'func'")
	}
	// Highlighted output should contain ANSI escape codes
	if !strings.Contains(got, "\x1b[") {
		t.Error("highlighted output should contain ANSI escape codes")
	}
}

func TestHighlightUnknownLang(t *testing.T) {
	code := `print("hello")`
	got := HighlightCode(code, "nonexistent-lang-xyz", testTheme())
	if got == "" {
		t.Fatal("HighlightCode returned empty string for unknown lang")
	}
	// Unknown lang should fall back to monochrome (no ANSI codes expected,
	// but the code should still be present)
	if !strings.Contains(got, `print("hello")`) {
		t.Error("fallback output should contain original code")
	}
}
