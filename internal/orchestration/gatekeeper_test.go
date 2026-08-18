package orchestration

import (
	"errors"
	"testing"
)

// ─── Task 4.3: RED — Test gatekeeper validation ───

func TestValidateSuccess(t *testing.T) {
	gk := NewGatekeeper(1)
	result := &SpawnResult{
		Output: "task completed successfully",
		Error:  nil,
	}

	err := gk.Validate(result)
	if err != nil {
		t.Errorf("Validate() returned unexpected error: %v", err)
	}
}

func TestValidateEmptyOutput(t *testing.T) {
	gk := NewGatekeeper(1)
	result := &SpawnResult{
		Output: "",
		Error:  nil,
	}

	err := gk.Validate(result)
	if err == nil {
		t.Error("Validate() should return error for empty output")
	}
	if !errors.Is(err, ErrEmptyOutput) {
		t.Errorf("Validate() error = %v, want ErrEmptyOutput", err)
	}
}

func TestValidateWithError(t *testing.T) {
	gk := NewGatekeeper(1)
	result := &SpawnResult{
		Output: "some output",
		Error:  errors.New("agent crashed"),
	}

	err := gk.Validate(result)
	if err == nil {
		t.Error("Validate() should return error when SpawnResult.Error is set")
	}
	if !errors.Is(err, ErrAgentError) {
		t.Errorf("Validate() error = %v, want ErrAgentError", err)
	}
}

func TestShouldRetry(t *testing.T) {
	gk := NewGatekeeper(3)
	result := &SpawnResult{
		Output: "",
		Error:  errors.New("failed"),
	}

	tests := []struct {
		name   string
		attempt int
		want   bool
	}{
		{"first attempt", 0, true},
		{"second attempt", 1, true},
		{"third attempt", 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gk.ShouldRetry(result, tt.attempt)
			if got != tt.want {
				t.Errorf("ShouldRetry(attempt=%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestShouldNotRetry(t *testing.T) {
	gk := NewGatekeeper(2)

	t.Run("exceeded max retries", func(t *testing.T) {
		result := &SpawnResult{Error: errors.New("failed")}
		got := gk.ShouldRetry(result, 2)
		if got {
			t.Error("ShouldRetry(attempt=2) = true, want false (max retries = 2)")
		}
	})

	t.Run("no error to retry", func(t *testing.T) {
		result := &SpawnResult{Error: nil}
		got := gk.ShouldRetry(result, 0)
		if got {
			t.Error("ShouldRetry with nil error should return false")
		}
	})
}
