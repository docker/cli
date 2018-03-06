package container

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/gotestyourself/gotestyourself/assert"
)

func TestCalculateMemUsageUnixNoCache(t *testing.T) {
	// Given
	stats := types.MemoryStats{Usage: 500, Stats: map[string]uint64{"cache": 400}}

	// When
	result := calculateMemUsageUnixNoCache(stats)

	// Then
	assert.Assert(t, inDelta(100.0, result, 1e-6))
}

func TestCalculateMemPercentUnixNoCache(t *testing.T) {
	// Given
	someLimit := float64(100.0)
	noLimit := float64(0.0)
	used := float64(70.0)

	// When and Then
	t.Run("Limit is set", func(t *testing.T) {
		result := calculateMemPercentUnixNoCache(someLimit, used)
		assert.Assert(t, inDelta(70.0, result, 1e-6))
	})
	t.Run("No limit, no cgroup data", func(t *testing.T) {
		result := calculateMemPercentUnixNoCache(noLimit, used)
		assert.Assert(t, inDelta(0.0, result, 1e-6))
	})
}

func inDelta(x, y, delta float64) func() (bool, string) {
	return func() (bool, string) {
		diff := x - y
		if diff < -delta || diff > delta {
			return false, fmt.Sprintf("%f != %f within %f", x, y, delta)
		}
		return true, ""
	}
}
