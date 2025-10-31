package image

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/icmd"
)

func TestPushQuietErrors(t *testing.T) {
	result := icmd.RunCmd(icmd.Command("docker", "push", "--quiet", "nosuchimage"))
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
	})
	assert.Check(t, is.Contains(result.Stderr(), "does not exist"))
	assert.Check(t, is.Contains(result.Stderr(), "nosuchimage"))
}
