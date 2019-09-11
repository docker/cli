package container

import (
	"testing"

	"github.com/docker/docker/api/types"
)

func TestCalculateBlockIO(t *testing.T) {
	blkio := types.BlkioStats{
		IoServiceBytesRecursive: []types.BlkioStatEntry{
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
	}
	blkRead, blkWrite := calculateBlockIO(blkio)
	if blkRead != 5815 {
		t.Fatalf("blkRead = %d, want 5815", blkRead)
	}
	if blkWrite != 593 {
		t.Fatalf("blkWrite = %d, want 593", blkWrite)
	}
}
