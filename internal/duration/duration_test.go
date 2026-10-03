package duration

import (
	"testing"
	"time"
)

func TestHumanDurationRounding(t *testing.T) {
	// 3.958 days should round to 4 days (Desktop-style), not floor to 3.
	d := time.Duration(3.958333 * float64(24*time.Hour))
	got := HumanDuration(d)
	if got != "4 days" {
		t.Fatalf("got %q, want 4 days", got)
	}
	// 2 weeks + 4 days (~18 days) closer to 3 weeks than 2.
	d = 2*week + 4*day
	got = HumanDuration(d)
	if got != "3 weeks" {
		t.Fatalf("got %q, want 3 weeks", got)
	}
}
