// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.25

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

// roundDuration returns d divided by unit, rounded to nearest integer.
func roundDuration(d, unit time.Duration) int {
	return int(float64(d)/float64(unit) + 0.5)
}

// HumanDuration returns a human-readable approximation of a duration
// (e.g. "About a minute", "4 hours ago", etc.) with consistent rounding
// at all unit boundaries.
//
// This is a drop-in replacement for docker/go-units HumanDuration that
// rounds day/week/month/year transitions instead of flooring them,
// matching Docker Desktop CREATED output behavior.
//
// Fixes docker/cli#6891.
func HumanDuration(d time.Duration) string {
	if seconds := int(d.Seconds()); seconds <= 0 {
		return "Less than a second"
	} else if seconds < 2 {
		return "1 second"
	} else if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	} else if minutes := int(d.Seconds()) / 60; minutes == 1 {
		return "About a minute"
	} else if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	} else if hours := roundDuration(d, time.Hour); hours == 1 {
		return "About an hour"
	} else if hours < 48 {
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
