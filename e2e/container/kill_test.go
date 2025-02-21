package container

import (
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/v28/e2e/internal/fixtures"
	"gotest.tools/v3/icmd"
	"gotest.tools/v3/poll"
)

func TestKillContainer(t *testing.T) {
	result := icmd.RunCommand("docker", "run", "-d", fixtures.AlpineImage, "top")
	result.Assert(t, icmd.Success)

	containerID := strings.TrimSpace(result.Stdout())

	// Kill with SIGTERM should kill the process
	result = icmd.RunCmd(icmd.Command("docker", "kill", "-s", "SIGTERM", containerID))

	result.Assert(t, icmd.Success)
	poll.WaitOn(t, containerStatus(t, containerID, "exited"), poll.WithDelay(100*time.Millisecond), poll.WithTimeout(5*time.Second))

	// Kill on a stop container should return an error
	result = icmd.RunCmd(icmd.Command("docker", "kill", containerID))
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "is not running",
	})
}

func containerStatus(t *testing.T, containerID, status string) func(poll.LogT) poll.Result {
	t.Helper()
	return func(poll.LogT) poll.Result {
		result := icmd.RunCommand("docker", "inspect", "-f", "{{ .State.Status }}", containerID)
		result.Assert(t, icmd.Success)
		actual := strings.TrimSpace(result.Stdout())
		if actual == status {
			return poll.Success()
		}
		return poll.Continue("expected status %s != %s", status, actual)
	}
}
