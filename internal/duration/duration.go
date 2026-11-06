// Package duration provides human-readable duration formatting for the CLI.
package duration

import (
	"fmt"
	"time"
)

const (
	day   = 24 * time.Hour
	week  = 7 * day
	month = 30 * day
	year  = 365 * day
)

// HumanDuration returns a human-readable approximation of a duration
// (e.g. "About a minute", "4 hours ago" style inputs without the "ago").
// Day/week/month/year boundaries use round-half-up so CLI CREATED columns
// stay closer to Docker Desktop.
func HumanDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	if seconds := int(d.Seconds()); seconds < 1 {
		return "Less than a second"
	} else if seconds == 1 {
		return "1 second"
	} else if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	} else if minutes := int(d.Minutes()); minutes == 1 {
		return "About a minute"
	} else if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	} else if hours := roundDuration(d, time.Hour); hours == 1 {
		return "About an hour"
	} else if d < 48*time.Hour {
		return fmt.Sprintf("%d hours", hours)
	} else if d < 2*week {
		return fmt.Sprintf("%d days", roundDuration(d, day))
	} else if d < 2*month {
		return fmt.Sprintf("%d weeks", roundDuration(d, week))
	} else if d < 2*year {
		return fmt.Sprintf("%d months", roundDuration(d, month))
	}
	return fmt.Sprintf("%d years", roundDuration(d, year))
}

func roundDuration(d, unit time.Duration) int {
	return int((d + unit/2) / unit)
}
