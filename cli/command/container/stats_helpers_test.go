package container

import (
	"fmt"
	"testing"

	"github.com/moby/moby/api/types/container"
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

func TestSortStats(t *testing.T) {
	entries := func() []StatsEntry {
		return []StatsEntry{
			{Container: "a", ID: "3", Name: "web", CPUPercentage: 10, Memory: 300, MemoryPercentage: 30, NetworkRx: 10, NetworkTx: 30, BlockRead: 20, BlockWrite: 10, PidsCurrent: 2},
			{Container: "b", ID: "2", Name: "db", CPUPercentage: 50, Memory: 100, MemoryPercentage: 10, NetworkRx: 50, NetworkTx: 10, BlockRead: 10, BlockWrite: 50, PidsCurrent: 9},
			{Container: "c", ID: "1", Name: "cache", CPUPercentage: 10, Memory: 200, MemoryPercentage: 20, NetworkRx: 20, NetworkTx: 20, BlockRead: 30, BlockWrite: 20, PidsCurrent: 5},
		}
	}
	containers := func(s []StatsEntry) []string {
		out := make([]string, 0, len(s))
		for _, e := range s {
			out = append(out, e.Container)
		}
		return out
	}

	for key, expected := range map[string][]string{
		"":        {"a", "b", "c"}, // unsorted: original order
		"cpu":     {"b", "a", "c"}, // descending, tie broken by container
		"mem":     {"a", "c", "b"},
		"mem%":    {"a", "c", "b"},
		"net":     {"b", "a", "c"},
		"block":   {"b", "c", "a"},
		"pids":    {"b", "c", "a"},
		"name":    {"c", "b", "a"}, // ascending
		"id":      {"c", "b", "a"}, // ascending
		"CPU":     {"b", "a", "c"}, // case-insensitive
	} {
		t.Run(key, func(t *testing.T) {
			s := entries()
			sortStats(s, key)
			assert.DeepEqual(t, containers(s), expected)
		})
	}
}
