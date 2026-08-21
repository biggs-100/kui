package util

import (
	"testing"
	"time"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want string
	}{
		{"zero", 0, "0"},
		{"small", 123, "123"},
		{"thousands", 1234, "1,234"},
		{"millions", 1234567, "1,234,567"},
		{"large", 1000000, "1,000,000"},
		{"negative", -1234, "-1,234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatNumber(tt.in); got != tt.want {
				t.Errorf("FormatNumber(%d) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatMoney(t *testing.T) {
	if got := FormatMoney(0); got != "$0.00" {
		t.Errorf("FormatMoney(0) = %q, want $0.00", got)
	}
	if got := FormatMoney(1234.5); got != "$1,234.50" {
		t.Errorf("FormatMoney(1234.5) = %q, want $1,234.50", got)
	}
	if got := FormatMoney(0.05); got != "$0.05" {
		t.Errorf("FormatMoney(0.05) = %q, want $0.05", got)
	}
}

func TestTodayTimeOrDateTime(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 30, 0, 0, now.Location())
	// Today should show time only
	got := TodayTimeOrDateTime(today, now)
	if got == "" {
		t.Error("TodayTimeOrDateTime returned empty for today")
	}
	// Yesterday should show date
	yesterday := today.AddDate(0, 0, -1)
	got2 := TodayTimeOrDateTime(yesterday, now)
	if got2 == "" {
		t.Error("TodayTimeOrDateTime returned empty for yesterday")
	}
	if got == got2 {
		t.Error("today and yesterday should produce different formats")
	}
}

func TestFormatDuration(t *testing.T) {
	if got := FormatDuration(0); got != "0s" {
		t.Errorf("FormatDuration(0) = %q, want 0s", got)
	}
	if got := FormatDuration(65 * time.Second); got != "1m 5s" {
		t.Errorf("FormatDuration(65s) = %q, want 1m 5s", got)
	}
	if got := FormatDuration(2*time.Hour + 30*time.Minute); got != "2h 30m" {
		t.Errorf("FormatDuration(2h30m) = %q, want 2h 30m", got)
	}
}
