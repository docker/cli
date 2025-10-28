package image

import (
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/icmd"
)

const registryPrefix = "registry:5000"

func TestPullQuiet(t *testing.T) {
	result := icmd.RunCommand("docker", "pull", "--quiet", fixtures.AlpineImage)
	result.Assert(t, icmd.Success)
	assert.Check(t, is.Equal(result.Stdout(), registryPrefix+"/alpine:frozen\n"))
	assert.Check(t, is.Equal(result.Stderr(), ""))
}
