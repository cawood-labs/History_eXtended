package search

import (
	"fmt"
	"time"
)

// RelTime formats startedAt (Unix seconds) as a short relative time string.
func RelTime(startedAt float64) string {
	if startedAt <= 0 {
		return "-"
	}
	sec := int64(startedAt)
	if sec > 1_000_000_000_000 {
		sec /= 1000
	}
	t := time.Unix(sec, 0)
	d := time.Since(t)
	if d < 0 {
		return "now"
	}
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}
