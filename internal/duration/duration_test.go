// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.25

package duration

import (
	"testing"
	"time"
)

func TestHumanDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		// Sub-second and seconds
		{name: "negative", duration: -1 * time.Second, want: "Less than a second"},
		{name: "zero", duration: 0, want: "Less than a second"},
		{name: "one second", duration: time.Second, want: "1 second"},
		{name: "30 seconds", duration: 30 * time.Second, want: "30 seconds"},

		// Minutes
		{name: "one minute", duration: time.Minute, want: "About a minute"},
		{name: "90 seconds", duration: 90 * time.Second, want: "About a minute"},
		{name: "5 minutes", duration: 5 * time.Minute, want: "5 minutes"},

		// Hours (already rounded in original)
		{name: "one hour", duration: time.Hour, want: "About an hour"},
		{name: "5 hours", duration: 5 * time.Hour, want: "5 hours"},
		{name: "47 hours", duration: 47 * time.Hour, want: "47 hours"},

		// Days (threshold: 48h to <336h) — rounding FIX
		{name: "2 days exact", duration: 2 * day, want: "2 days"},
		{name: "13.5 days rounds up to 14", duration: 324 * time.Hour, want: "14 days"},
		{name: "6.5 days rounds up to 7", duration: 156 * time.Hour, want: "7 days"},
		{name: "6.4 days rounds down to 6", duration: 154 * time.Hour, want: "6 days"},

		// Weeks (threshold: 336h to <1440h) — rounding FIX
		{name: "2 weeks exact", duration: 2 * week, want: "2 weeks"},
		{name: "2.6 weeks rounds up to 3", duration: 18*day + 6*time.Hour, want: "3 weeks"},
		{name: "2.4 weeks rounds down to 2", duration: 17*day - 6*time.Hour, want: "2 weeks"},

		// Months (threshold: 1440h to <17520h) — rounding FIX
		{name: "2 months exact", duration: 2 * month, want: "2 months"},
		{name: "2.6 months rounds up to 3", duration: 78 * day, want: "3 months"},
		{name: "2.4 months rounds down to 2", duration: 72 * day, want: "2 months"},

		// Years (threshold: >=17520h) — rounding FIX
		{name: "2 years exact", duration: 2 * year, want: "2 years"},
		{name: "2.6 years rounds up to 3", duration: 949 * day, want: "3 years"},
		{name: "2.4 years rounds down to 2", duration: 876 * day, want: "2 years"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HumanDuration(tt.duration)
			if got != tt.want {
				t.Errorf("HumanDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestRoundDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		unit time.Duration
		want int
	}{
		{name: "exact", d: 2 * day, unit: day, want: 2},
		{name: "round up", d: 36 * time.Hour, unit: day, want: 2},
		{name: "round down", d: 30 * time.Hour, unit: day, want: 1},
		{name: "half up", d: 12 * time.Hour, unit: day, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roundDuration(tt.d, tt.unit)
			if got != tt.want {
				t.Errorf("roundDuration(%v, %v) = %d, want %d", tt.d, tt.unit, got, tt.want)
			}
		})
	}
}
