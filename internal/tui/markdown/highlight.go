package markdown

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/biggs-100/kui/internal/tui/theme"
)

// HighlightCode applies Chroma-based syntax highlighting to code.
// Falls back to monochrome if the language is unknown.
func HighlightCode(code string, lang string, t *theme.Theme) string {
	// Try to find a lexer for the language
	lexer := lexers.Get(lang)
	if lexer == nil {
		// Fallback: return plain text
		return code
	}

	// Build a Chroma style from the theme's syntax colors
	s := buildChromaStyle(t)

	// Format with ANSI terminal output
	formatter := formatters.TTY16m
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	var b strings.Builder
	if err := formatter.Format(&b, s, iterator); err != nil {
		return code
	}

	return b.String()
}

// buildChromaStyle creates a Chroma style from the theme's syntax colors.
func buildChromaStyle(t *theme.Theme) *chroma.Style {
	entries := chroma.StyleEntries{
		chroma.CommentSingle:      t.SyntaxComment,
		chroma.CommentMultiline:   t.SyntaxComment,
		chroma.Comment:            t.SyntaxComment,
		chroma.Keyword:            t.SyntaxKeyword,
		chroma.KeywordNamespace:   t.SyntaxKeyword,
		chroma.KeywordDeclaration: t.SyntaxKeyword,
		chroma.NameFunction:       t.SyntaxFunction,
		chroma.NameBuiltin:        t.SyntaxFunction,
		chroma.LiteralString:      t.SyntaxString,
		chroma.LiteralStringDouble: t.SyntaxString,
		chroma.LiteralNumber:      t.SyntaxNumber,
		chroma.LiteralNumberFloat: t.SyntaxNumber,
		chroma.LiteralNumberInteger: t.SyntaxNumber,
		chroma.NameClass:          t.SyntaxType,
		chroma.NameVariable:       t.SyntaxVariable,
		chroma.NameBuiltinPseudo:  t.SyntaxVariable,
		chroma.GenericDeleted:     t.Error,
		chroma.GenericInserted:    t.Success,
	}

	style, err := chroma.NewStyle("kui-custom", entries)
	if err != nil {
		// Fallback to monokai if build fails
		return styles.Get("monokai")
	}
	return style
}
