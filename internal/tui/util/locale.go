package util

import (
	"fmt"
	"time"
)

func FormatNumber(n int) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var out []byte
	for i, c := range s {
		if i != 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return sign + string(out)
}

func FormatMoney(v float64) string {
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	cents := int(v*100 + 0.5)
	return fmt.Sprintf("%s$%s.%02d", sign, FormatNumber(cents/100), cents%100)
}

func TodayTimeOrDateTime(t, now time.Time) string {
	if t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day() {
		return t.Format("15:04")
	}
	return t.Format("2006-01-02")
}

func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs == 0 {
			return fmt.Sprintf("%dm", mins)
		}
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}
