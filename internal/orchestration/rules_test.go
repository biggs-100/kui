package orchestration

import (
	"testing"
)

// ─── Task 4.1: RED — Test delegation rules ───

func TestDefaultRules(t *testing.T) {
	r := DefaultRules()

	if r.ExploreThreshold != 4 {
		t.Errorf("ExploreThreshold = %d, want 4", r.ExploreThreshold)
	}
	if r.WriteThreshold != 2 {
		t.Errorf("WriteThreshold = %d, want 2", r.WriteThreshold)
	}
	if !r.ContextRule {
		t.Error("ContextRule = false, want true")
	}
}

func TestShouldDelegateExplore(t *testing.T) {
	tests := []struct {
		name      string
		threshold int
		fileCount int
		want      bool
	}{
		{"at threshold", 4, 4, true},
		{"above threshold", 4, 5, true},
		{"well above threshold", 4, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Rules{ExploreThreshold: tt.threshold, WriteThreshold: 2, ContextRule: true}
			got := r.ShouldDelegate("explore", tt.fileCount)
			if got != tt.want {
				t.Errorf("ShouldDelegate(explore, %d) = %v, want %v", tt.fileCount, got, tt.want)
			}
		})
	}
}

func TestShouldDelegateWrite(t *testing.T) {
	tests := []struct {
		name      string
		threshold int
		fileCount int
		want      bool
	}{
		{"at threshold", 2, 2, true},
		{"above threshold", 2, 3, true},
		{"well above threshold", 2, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Rules{ExploreThreshold: 4, WriteThreshold: tt.threshold, ContextRule: true}
			got := r.ShouldDelegate("write", tt.fileCount)
			if got != tt.want {
				t.Errorf("ShouldDelegate(write, %d) = %v, want %v", tt.fileCount, got, tt.want)
			}
		})
	}
}

func TestShouldDelegateContext(t *testing.T) {
	tests := []struct {
		name        string
		contextRule bool
		want        bool
	}{
		{"enabled", true, true},
		{"disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Rules{ExploreThreshold: 4, WriteThreshold: 2, ContextRule: tt.contextRule}
			got := r.ShouldDelegate("context", 0)
			if got != tt.want {
				t.Errorf("ShouldDelegate(context, 0) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldNotDelegate(t *testing.T) {
	tests := []struct {
		name      string
		action    string
		fileCount int
	}{
		{"explore below threshold", "explore", 3},
		{"write below threshold", "write", 1},
		{"unknown action", "unknown", 100},
		{"explore at zero", "explore", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := DefaultRules()
			got := r.ShouldDelegate(tt.action, tt.fileCount)
			if got {
				t.Errorf("ShouldDelegate(%q, %d) = true, want false", tt.action, tt.fileCount)
			}
		})
	}
}
