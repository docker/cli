package image

import (
	"testing"

	"gotest.tools/v3/icmd"
)

func TestPushQuietErrors(t *testing.T) {
	result := icmd.RunCmd(icmd.Command("docker", "push", "--quiet", "nosuchimage"))
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "An image does not exist locally with the tag: nosuchimage",
	})
}
