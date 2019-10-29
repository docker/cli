package container

import (
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/kr/pty"
	"gotest.tools/assert"
	"gotest.tools/icmd"
	"gotest.tools/poll"
)

// TestSigProxyWithTTY tests that killing the docker CLI forwards the signal to
// the container, and kills the container's process. Test-case for moby/moby#28872
func TestSigProxyWithTTY(t *testing.T) {
	cmd := exec.Command("docker", "run", "-i", "-t", "--init", "--name", t.Name(), fixtures.BusyboxImage, "sleep", "30")
	p, err := pty.Start(cmd)
	defer func() {
		_ = cmd.Wait()
		_ = p.Close()
	}()
	assert.NilError(t, err, "failed to start container")
	defer icmd.RunCommand("docker", "container", "rm", "-f", t.Name())

	poll.WaitOn(t, containerExistsWithStatus(t.Name(), "running"), poll.WithDelay(100*time.Millisecond), poll.WithTimeout(5*time.Second))

	pid := cmd.Process.Pid
	t.Logf("terminating PID %d", pid)
	err = syscall.Kill(pid, syscall.SIGTERM)
	assert.NilError(t, err)

	poll.WaitOn(t, containerExistsWithStatus(t.Name(), "exited"), poll.WithDelay(100*time.Millisecond), poll.WithTimeout(5*time.Second))
}

func containerExistsWithStatus(containerID, status string) func(poll.LogT) poll.Result {
	return func(poll.LogT) poll.Result {
		result := icmd.RunCommand("docker", "inspect", "-f", "{{ .State.Status }}", containerID)
		// ignore initial failures as the container may not yet exist (i.e., don't result.Assert(t, icmd.Success))

		actual := strings.TrimSpace(result.Stdout())
		if actual == status {
			return poll.Success()
		}
		return poll.Continue("expected status %s != %s", status, actual)
	}
}
