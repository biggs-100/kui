package theme

import (
	"fmt"
	"strings"
)

// Tint blends background and foreground hex colors with weight a (0..1).
func Tint(bg, fg string, a float64) string {
	if a <= 0 {
		if isHex(bg) {
			return strings.ToLower(bg)
		}
		return bg
	}
	if a >= 1 {
		if isHex(fg) {
			return strings.ToLower(fg)
		}
		return fg
	}
	br, bgG, bb, ok1 := parseHex(bg)
	fr, fgG, fb, ok2 := parseHex(fg)
	if !ok1 || !ok2 {
		if ok1 {
			return strings.ToLower(bg)
		}
		if ok2 {
			return strings.ToLower(fg)
		}
		return bg
	}
	r := uint8(float64(br)*(1-a) + float64(fr)*a + 0.5)
	g := uint8(float64(bgG)*(1-a) + float64(fgG)*a + 0.5)
	b := uint8(float64(bb)*(1-a) + float64(fb)*a + 0.5)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// GetSyntaxRules returns a map from token type to theme color.
func GetSyntaxRules(t *Theme) map[string]string {
	if t == nil {
		return map[string]string{}
	}
	return map[string]string{
		"comment": t.SyntaxComment, "keyword": t.SyntaxKeyword, "function": t.SyntaxFunction,
		"string": t.SyntaxString, "number": t.SyntaxNumber, "type": t.SyntaxType,
		"variable": t.SyntaxVariable, "operator": t.SyntaxOperator, "punctuation": t.SyntaxPunctuation,
	}
}

// SelectedForeground returns the color for selected list items.
func SelectedForeground(t *Theme) string {
	if t == nil {
		return ""
	}
	if t.SelectedListItemText != "" {
		return t.SelectedListItemText
	}
	if t.Text != "" {
		return t.Text
	}
	return t.FG
}

func isHex(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func parseHex(s string) (r, g, b uint8, ok bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	var ri, gi, bi int
	_, err := fmt.Sscanf(s, "%02x%02x%02x", &ri, &gi, &bi)
	if err != nil {
		return 0, 0, 0, false
	}
	return uint8(ri), uint8(gi), uint8(bi), true
}
