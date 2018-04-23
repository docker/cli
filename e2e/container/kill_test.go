package container

import (
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/gotestyourself/gotestyourself/icmd"
	"github.com/gotestyourself/gotestyourself/poll"
)

func TestKillContainer(t *testing.T) {
	containerID := runBackgroundTop(t)

	// Kill with SIGTERM should kill the process
	result := icmd.RunCmd(
		icmd.Command("docker", "kill", "-s", "SIGTERM", containerID),
	)

	result.Assert(t, icmd.Success)
	poll.WaitOn(t, containerStatus(t, containerID, "exited"), poll.WithDelay(100*time.Millisecond), poll.WithTimeout(5*time.Second))

	// Kill on a stop container should return an error
	result = icmd.RunCmd(
		icmd.Command("docker", "kill", containerID),
	)
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "is not running",
	})
}

func runBackgroundTop(t *testing.T) string {
	result := icmd.RunCommand("docker", "run", "-d", fixtures.AlpineImage, "top")
	result.Assert(t, icmd.Success)
	return strings.TrimSpace(result.Stdout())
}

func containerStatus(t *testing.T, containerID, status string) func(poll.LogT) poll.Result {
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
