package container // import "docker.com/cli/v28/cli/command/container"

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/container"
	"gotest.tools/v3/assert"
)

func TestCalculateMemUsageUnixNoCache(t *testing.T) {
	result := calculateMemUsageUnixNoCache(container.MemoryStats{Usage: 500, Stats: map[string]uint64{"total_inactive_file": 400}})
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

func TestCalculateBlockIO(t *testing.T) {
	blkRead, blkWrite := calculateBlockIO(container.BlkioStats{
		IoServiceBytesRecursive: []container.BlkioStatEntry{
			{Major: 8, Minor: 0, Op: "read", Value: 1234},
			{Major: 8, Minor: 1, Op: "read", Value: 4567},
			{Major: 8, Minor: 0, Op: "Read", Value: 6},
			{Major: 8, Minor: 1, Op: "Read", Value: 8},
			{Major: 8, Minor: 0, Op: "write", Value: 123},
			{Major: 8, Minor: 1, Op: "write", Value: 456},
			{Major: 8, Minor: 0, Op: "Write", Value: 6},
			{Major: 8, Minor: 1, Op: "Write", Value: 8},
			{Major: 8, Minor: 1, Op: "", Value: 456},
		},
	})
	if blkRead != 5815 {
		t.Fatalf("blkRead = %d, want 5815", blkRead)
	}
	if blkWrite != 593 {
		t.Fatalf("blkWrite = %d, want 593", blkWrite)
	}
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
