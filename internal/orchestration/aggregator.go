package orchestration

import (
	"strings"
)

// ResultAggregator combines results from multiple agents.
type ResultAggregator struct{}

// Aggregate combines multiple spawn results based on the given mode.
// Modes: "merge" (concatenate with separators), "summary" (join outputs),
// "select" (pick longest output). Unknown modes default to merge.
func (r *ResultAggregator) Aggregate(results []*SpawnResult, mode string) string {
	if len(results) == 0 {
		return "(no results)"
	}

	if len(results) == 1 {
		return results[0].Output
	}

	switch mode {
	case "merge":
		return r.merge(results)
	case "summary":
		return r.summary(results)
	case "select":
		return r.selectBest(results)
	default:
		return r.merge(results)
	}
}

// merge concatenates all outputs with separators.
func (r *ResultAggregator) merge(results []*SpawnResult) string {
	var sb strings.Builder
	for i, result := range results {
		if result.Output == "" {
			continue
		}
		if i > 0 {
			sb.WriteString("\n---\n")
		}
		sb.WriteString(result.Output)
	}
	return sb.String()
}

// summary joins all outputs into a combined summary.
func (r *ResultAggregator) summary(results []*SpawnResult) string {
	var sb strings.Builder
	for i, result := range results {
		if result.Output == "" {
			continue
		}
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(result.Output)
	}
	return sb.String()
}

// selectBest picks the result with the longest output.
func (r *ResultAggregator) selectBest(results []*SpawnResult) string {
	best := ""
	for _, result := range results {
		if len(result.Output) > len(best) {
			best = result.Output
		}
	}
	return best
}
