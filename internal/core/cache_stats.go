package core

import (
	"fmt"
	"io"
	"sync"
)

// CacheStats tracks cumulative token usage and cache hits across a session.
type CacheStats struct {
	mu            sync.Mutex
	totalInput    int
	totalOutput   int
	totalCached   int
	totalRequests int
}

// Record adds a usage sample to the stats.
func (s *CacheStats) Record(usage Usage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalInput += usage.InputTokens
	s.totalOutput += usage.OutputTokens
	s.totalCached += usage.CachedTokens
	s.totalRequests++
}

// HitRatio returns the cache hit ratio (0.0–1.0). Returns 0 when no input tokens.
func (s *CacheStats) HitRatio() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.totalInput == 0 {
		return 0
	}
	return float64(s.totalCached) / float64(s.totalInput)
}

// Summary returns a one-line cache stats summary.
func (s *CacheStats) Summary() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.totalRequests == 0 {
		return "cache: no requests yet"
	}
	ratio := float64(s.totalCached) / float64(s.totalInput) * 100
	saved := float64(s.totalCached) * 0.9 // cache reads cost 0.1x
	return fmt.Sprintf("cache: %d/%d input tokens cached (%.0f%% hit, ~%.0f%% cost saved) across %d requests",
		s.totalCached, s.totalInput, ratio, saved, s.totalRequests)
}

// CacheStatsObserver implements UsageObserver and writes cache stats to a writer.
type CacheStatsObserver struct {
	stats *CacheStats
	w     io.Writer
}

// NewCacheStatsObserver creates an observer that writes cache stats to w.
func NewCacheStatsObserver(w io.Writer) *CacheStatsObserver {
	return &CacheStatsObserver{
		stats: &CacheStats{},
		w:     w,
	}
}

// OnUsage records usage and writes a summary line.
func (o *CacheStatsObserver) OnUsage(usage Usage) {
	o.stats.Record(usage)
	if usage.CachedTokens > 0 {
		fmt.Fprintln(o.w, o.stats.Summary())
	}
}

// OnTurnStart is a no-op (required by Observer interface).
func (o *CacheStatsObserver) OnTurnStart() {}

// OnTurnEnd is a no-op (required by Observer interface).
func (o *CacheStatsObserver) OnTurnEnd() {}

// OnToolCall is a no-op (required by Observer interface).
func (o *CacheStatsObserver) OnToolCall(call ToolCall) {}

// OnToolResult is a no-op (required by Observer interface).
func (o *CacheStatsObserver) OnToolResult(callID, result string) {}

// Stats returns the underlying stats for querying.
func (o *CacheStatsObserver) Stats() *CacheStats {
	return o.stats
}
