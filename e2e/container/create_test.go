package container

import (
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"gotest.tools/v3/icmd"
)

func TestCreateWithEmptySourceVolume(t *testing.T) {
	icmd.RunCmd(icmd.Command("docker", "create", "-v", ":/volume", fixtures.AlpineImage)).
		Assert(t, icmd.Expected{
			ExitCode: 125,
			Err:      "empty section between colons",
		})
}

func TestCreateWithEmptyVolumeSpec(t *testing.T) {
	icmd.RunCmd(icmd.Command("docker", "create", "-v", "", fixtures.AlpineImage)).
		Assert(t, icmd.Expected{
			ExitCode: 125,
			Err:      "invalid empty volume spec",
		})
}
