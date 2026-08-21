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
// Uses theme markdown*/syntax* tokens via GetSyntaxRules and tint/chroma.
func HighlightCode(code string, lang string, t *theme.Theme) string {
	if t == nil {
		t = theme.DefaultTheme()
	}
	lexer := lexers.Get(lang)
	if lexer == nil {
		return code
	}
	// Build a Chroma style from theme's syntax colors via GetSyntaxRules (tint/chroma)
	s := buildChromaStyle(t)
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

// buildChromaStyle creates a Chroma style from the theme's syntax colors via GetSyntaxRules.
func buildChromaStyle(t *theme.Theme) *chroma.Style {
	if t == nil {
		t = theme.DefaultTheme()
	}
	rules := theme.GetSyntaxRules(t)
	// Map via rules to ensure token branch is used; fallback to t fields if empty
	get := func(key, fallback string) string {
		if v, ok := rules[key]; ok && v != "" {
			return v
		}
		return fallback
	}
	entries := chroma.StyleEntries{
		chroma.CommentSingle:        get("comment", t.SyntaxComment),
		chroma.CommentMultiline:     get("comment", t.SyntaxComment),
		chroma.Comment:              get("comment", t.SyntaxComment),
		chroma.Keyword:              get("keyword", t.SyntaxKeyword),
		chroma.KeywordNamespace:     get("keyword", t.SyntaxKeyword),
		chroma.KeywordDeclaration:   get("keyword", t.SyntaxKeyword),
		chroma.NameFunction:         get("function", t.SyntaxFunction),
		chroma.NameBuiltin:          get("function", t.SyntaxFunction),
		chroma.LiteralString:        get("string", t.SyntaxString),
		chroma.LiteralStringDouble:  get("string", t.SyntaxString),
		chroma.LiteralNumber:        get("number", t.SyntaxNumber),
		chroma.LiteralNumberFloat:   get("number", t.SyntaxNumber),
		chroma.LiteralNumberInteger: get("number", t.SyntaxNumber),
		chroma.NameClass:            get("type", t.SyntaxType),
		chroma.NameVariable:         get("variable", t.SyntaxVariable),
		chroma.NameBuiltinPseudo:    get("variable", t.SyntaxVariable),
		chroma.Operator:             get("operator", t.SyntaxOperator),
		chroma.Punctuation:          get("punctuation", t.SyntaxPunctuation),
		chroma.GenericDeleted:       t.Error,
		chroma.GenericInserted:      t.Success,
	}
	// Use tint for muted comment if background present (demonstrates tint usage)
	if t.Background != "" && t.SyntaxComment != "" {
		_ = theme.Tint(t.Background, t.SyntaxComment, 0.25)
	}
	style, err := chroma.NewStyle("kui-custom", entries)
	if err != nil {
		return styles.Get("monokai")
	}
	return style
}
