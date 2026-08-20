package tui

import (
	"regexp"
	"sort"
	"strings"
)

// splitRe splits tokens on whitespace and '/' as Pi does: /[\s/]+/.
var splitRe = regexp.MustCompile(`[\s/]+`)

// splitTokens splits query into tokens on whitespace and '/', lowercased, ignoring empties.
func splitTokens(query string) []string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	parts := splitRe.Split(q, -1)
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// isWordBoundaryChar reports whether c is a word boundary separator for scoring.
// Pi scoring gives -10 for matches at a word boundary.
func isWordBoundaryChar(c rune) bool {
	return c == ' ' || c == '/' || c == '-' || c == '_' || c == '.' || c == ':' || c == '\\'
}

// scoreToken scores a single token against lowercased text using sequential scan.
// Returns (matched, score). Score uses: -10 word boundary, -5 consecutive, +2 per gap, +0.1 index.
// If token == text exactly (case-insensitive), returns -100.
func scoreToken(token, lowerText string) (bool, int) {
	// exact handled outside; but keep quick path
	if token == lowerText {
		return true, -100
	}
	// Sequential scan to find positions of each token char in order.
	positions := make([]int, 0, len(token))
	ti := 0
	for i := 0; i < len(lowerText) && ti < len(token); i++ {
		if lowerText[i] == token[ti] {
			positions = append(positions, i)
			ti++
		}
	}
	if ti != len(token) {
		return false, 0
	}
	// Compute score as float then convert to int to preserve 0.1 granularity.
	score := 0.0
	// +0.1 per index of first match (lower index is better)
	score += float64(positions[0]) * 0.1
	for idx, pos := range positions {
		// -10 for word boundary
		isBoundary := false
		if pos == 0 {
			isBoundary = true
		} else {
			if isWordBoundaryChar(rune(lowerText[pos-1])) {
				isBoundary = true
			}
		}
		if isBoundary {
			score -= 10
		}
		if idx > 0 {
			gap := pos - positions[idx-1] - 1
			if gap == 0 {
				score -= 5 // consecutive bonus
			} else {
				score += float64(gap) * 2 // gap penalty
			}
		}
	}
	// Scale to int preserving 0.1 resolution: multiply by 10 and round
	return true, int(score * 10)
}

// fuzzyMatch reports whether query matches text using Pi token-sequential fuzzy logic.
// All tokens (split on /[\s/]+/) must match sequentially. Returns (matched, score)
// where lower score is better (more negative = better). Exact match yields -1000 (scaled).
func fuzzyMatch(query, text string) (bool, int) {
	qLower := strings.ToLower(strings.TrimSpace(query))
	tLower := strings.ToLower(text)

	if qLower == "" {
		return true, 0
	}
	// Exact case-insensitive match: strong bonus
	if qLower == tLower {
		return true, -1000 // -100 scaled by 10
	}
	tokens := splitTokens(qLower)
	if len(tokens) == 0 {
		// query was "/" or only separators -> match everything
		return true, 0
	}
	total := 0
	for _, tok := range tokens {
		ok, s := scoreToken(tok, tLower)
		if !ok {
			return false, 0
		}
		total += s
	}
	return true, total
}

// FuzzyMatch is an exported wrapper for fuzzyMatch.
func FuzzyMatchScore(query, text string) (bool, int) {
	return fuzzyMatch(query, text)
}

// fuzzyFilter filters and sorts items by fuzzyMatch score. Items with best (lowest) score first.
func fuzzyFilter[T any](query string, items []T, getText func(T) string) []T {
	type scored struct {
		item  T
		score int
		text  string
	}
	var scoredItems []scored
	for _, it := range items {
		txt := getText(it)
		ok, score := fuzzyMatch(query, txt)
		if ok {
			scoredItems = append(scoredItems, scored{item: it, score: score, text: txt})
		}
	}
	sort.Slice(scoredItems, func(i, j int) bool {
		if scoredItems[i].score != scoredItems[j].score {
			return scoredItems[i].score < scoredItems[j].score
		}
		return scoredItems[i].text < scoredItems[j].text
	})
	out := make([]T, len(scoredItems))
	for i, s := range scoredItems {
		out[i] = s.item
	}
	return out
}

// FuzzyFilter is an exported generic wrapper.
func FuzzyFilter[T any](query string, items []T, getText func(T) string) []T {
	return fuzzyFilter(query, items, getText)
}
