package markdown

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/biggs-100/kui/internal/tui/theme"
)

var (
	// Heading: # ... at start of line
	reHeading = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	// Bold: **...**
	reBold = regexp.MustCompile(`\*\*(.+?)\*\*`)
	// Italic: *...* (single asterisk, not inside **)
	reItalic = regexp.MustCompile(`\*([^*\n]+)\*`)
	// Inline code: `...`
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	// Fenced code block: ```lang\n...\n```
	reFence = regexp.MustCompile("(?s)```(\\w*)\\n(.*?)\\n```")
	// List item: - ...
	reListItem = regexp.MustCompile(`(?m)^-\s+(.+)$`)
	// Blockquote: > ...
	reBlockquote = regexp.MustCompile(`(?m)^>\s+(.+)$`)
)

// Render converts raw markdown content into lipgloss-styled string output.
// Unrecognized syntax passes through as plain text.
func Render(content string, styles *theme.Styles) string {
	if content == "" {
		return ""
	}

	// Process fenced code blocks first (they span multiple lines)
	result := reFence.ReplaceAllStringFunc(content, func(match string) string {
		parts := reFence.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		lang := parts[1]
		code := parts[2]
		// Style code block with theme backgroundElement
		codeBlockStyle := styles.CodeBlock
		if styles == nil {
			codeBlockStyle = lipgloss.NewStyle().Background(lipgloss.Color(theme.DefaultTheme().BackgroundElement)).Padding(0, 1)
		}
		var rendered string
		if lang != "" {
			highlighted := HighlightCode(code, lang, theme.DefaultTheme())
			rendered = codeBlockStyle.Render(highlighted)
		} else {
			if styles != nil {
				rendered = codeBlockStyle.Render(styles.ToolResult.Render(code))
			} else {
				rendered = codeBlockStyle.Render(code)
			}
		}
		return rendered
	})

	// Process block-level patterns (per line)
	lines := strings.Split(result, "\n")
	var out []string
	for _, line := range lines {
		// Thought: prefix styled with warning token
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Thought:") {
			// Preserve prefix styling with theme warning
			thoughtStyle := styles.Thought
			if styles == nil {
				thoughtStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.DefaultTheme().Warning)).Bold(true)
			}
			out = append(out, thoughtStyle.Render(line))
			continue
		}


		// Heading
		if m := reHeading.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			text := m[2]
			var styled string
			switch {
			case level <= 1:
				styled = styles.UserRole.Render(text) // H1 — bold primary
			case level == 2:
				styled = styles.ActiveTab.Render(text) // H2
			default:
				styled = styles.StatusLine.Render(text) // H3+
			}
			out = append(out, styled)
			continue
		}

		// Blockquote
		if m := reBlockquote.FindStringSubmatch(line); m != nil {
			quoted := styles.Hint.Render("│ " + m[1])
			out = append(out, quoted)
			continue
		}

		// List item
		if m := reListItem.FindStringSubmatch(line); m != nil {
			item := styles.Hint.Render("  • ") + applyInline(m[1], styles)
			out = append(out, item)
			continue
		}

		// Plain line — apply inline styling
		out = append(out, applyInline(line, styles))
	}

	return strings.Join(out, "\n")
}

// applyInline processes inline markdown patterns: bold, italic, inline code.
func applyInline(text string, styles *theme.Styles) string {
	// Inline code (must come before bold/italic to avoid conflicts)
	text = reInlineCode.ReplaceAllStringFunc(text, func(match string) string {
		m := reInlineCode.FindStringSubmatch(match)
		if len(m) < 2 {
			return match
		}
		return styles.ToolName.Render(m[1])
	})

	// Bold
	text = reBold.ReplaceAllStringFunc(text, func(match string) string {
		m := reBold.FindStringSubmatch(match)
		if len(m) < 2 {
			return match
		}
		return styles.UserRole.Render(m[1])
	})

	// Italic
	text = reItalic.ReplaceAllStringFunc(text, func(match string) string {
		m := reItalic.FindStringSubmatch(match)
		if len(m) < 2 {
			return match
		}
		return styles.Hint.Render(m[1])
	})

	return text
}
