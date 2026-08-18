package core

import (
	"bytes"
	"strings"
	"testing"
)

func TestCacheStatsRecord(t *testing.T) {
	stats := &CacheStats{}

	stats.Record(Usage{InputTokens: 1000, OutputTokens: 500, CachedTokens: 0})
	if stats.HitRatio() != 0 {
		t.Errorf("HitRatio after no cache = %f, want 0", stats.HitRatio())
	}

	stats.Record(Usage{InputTokens: 1000, OutputTokens: 500, CachedTokens: 800})
	if stats.HitRatio() != 0.4 { // 800/2000
		t.Errorf("HitRatio = %f, want 0.4", stats.HitRatio())
	}
}

func TestCacheStatsSummary(t *testing.T) {
	stats := &CacheStats{}

	if got := stats.Summary(); got != "cache: no requests yet" {
		t.Errorf("empty Summary = %q", got)
	}

	stats.Record(Usage{InputTokens: 1000, OutputTokens: 500, CachedTokens: 800})
	summary := stats.Summary()
	if !strings.Contains(summary, "800/1000") {
		t.Errorf("Summary doesn't contain token counts: %q", summary)
	}
	if !strings.Contains(summary, "1 requests") {
		t.Errorf("Summary doesn't contain request count: %q", summary)
	}
}

func TestCacheStatsObserver(t *testing.T) {
	var buf bytes.Buffer
	obs := NewCacheStatsObserver(&buf)

	// No cache hit — no output.
	obs.OnUsage(Usage{InputTokens: 1000, CachedTokens: 0})
	if buf.Len() != 0 {
		t.Errorf("expected no output for zero cache, got %q", buf.String())
	}

	// Cache hit — should write summary.
	obs.OnUsage(Usage{InputTokens: 1000, CachedTokens: 500})
	if buf.Len() == 0 {
		t.Error("expected output for cache hit, got none")
	}

	if obs.Stats().HitRatio() != 0.25 { // 500/2000
		t.Errorf("Stats().HitRatio() = %f, want 0.25", obs.Stats().HitRatio())
	}
}
