package containerized

import (
	"context"
	"testing"

	"github.com/containerd/containerd/containers"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"gotest.tools/assert"
)

func TestWithAllCapabilities(t *testing.T) {
	c := &containers.Container{}
	s := &specs.Spec{
		Process: &specs.Process{},
	}
	ctx := context.Background()
	err := WithAllCapabilities(ctx, nil, c, s)
	assert.NilError(t, err)
	assert.Assert(t, len(s.Process.Capabilities.Bounding) > 0)
}
