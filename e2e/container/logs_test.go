package container

import (
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/e2e/internal/fixtures"
	"gotest.tools/v3/icmd"
	"gotest.tools/v3/poll"
)

func TestLogsReattach(t *testing.T) {
	result := icmd.RunCommand("docker", "run", "-d", fixtures.AlpineImage,
		"sh", "-c", "echo hi; while true; do sleep 1; done")
	result.Assert(t, icmd.Success)
	containerID := strings.TrimSpace(result.Stdout())

	cmd := icmd.Command("docker", "logs", "-f", "-d", "5", containerID)
	// cmd := icmd.Command("docker", "logs", containerID)
	result = icmd.StartCmd(cmd)

	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		if strings.Contains(result.Stdout(), "hi") {
			return poll.Success()
		}
		return poll.Continue("waiting")
	}, poll.WithDelay(1*time.Second), poll.WithTimeout(5*time.Second))

	icmd.RunCommand("docker", "restart", containerID).Assert(t, icmd.Success)

	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		// if there is another "hi" then the container was successfully restarted,
		// printed "hi" again and `docker logs` stayed attached
		if strings.Contains(result.Stdout(), "hi\nhi") { //nolint:dupword
			return poll.Success()
		}
		return poll.Continue(result.Stdout())
	}, poll.WithDelay(1*time.Second), poll.WithTimeout(10*time.Second))

	icmd.RunCommand("docker", "stop", containerID).Assert(t, icmd.Success)

	icmd.WaitOnCmd(time.Second*10, result).Assert(t, icmd.Expected{
		ExitCode: 0,
	})
}
