package container

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/docker/cli/e2e/internal/fixtures"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
	"gotest.tools/v3/skip"
)

func TestAttachExitCode(t *testing.T) {
	const exitCode = 21
	result := icmd.RunCommand("docker", "run", "-d", "-i", "--rm", fixtures.AlpineImage, "sh", "-c", fmt.Sprintf("read; exit %d", exitCode))
	result.Assert(t, icmd.Success)

	containerID := strings.TrimSpace(result.Stdout())

	result = icmd.RunCmd(icmd.Command("docker", "attach", containerID), withStdinNewline)
	result.Assert(t, icmd.Expected{ExitCode: exitCode})
}

func withStdinNewline(cmd *icmd.Cmd) {
	cmd.Stdin = strings.NewReader("\n")
}

// Regression test for https://github.com/docker/cli/issues/5294
func TestAttachInterrupt(t *testing.T) {
	// this is a new test, it already did not work (inside dind) when over ssh
	// todo(laurazard): make this test work w/ dind over ssh
	skip.If(t, strings.Contains(os.Getenv("DOCKER_HOST"), "ssh://"))

	// if
	result := icmd.RunCommand("docker", "run", "-d", fixtures.AlpineImage,
		"sh", "-c", "trap \"exit 33\" SIGINT; for i in $(seq 100); do sleep 0.1; done; exit 34")
	result.Assert(t, icmd.Success)
	containerID := strings.TrimSpace(result.Stdout())

	// run it as such so we can signal it later
	c := exec.Command("docker", "attach", containerID)
	d := bytes.Buffer{}
	c.Stdout = &d
	c.Stderr = &d
	_, err := pty.Start(c)
	assert.NilError(t, err)

	// have to wait a bit to give time for the command to execute/print
	time.Sleep(500 * time.Millisecond)
	c.Process.Signal(os.Interrupt)

	_ = c.Wait()
	// the CLI should exit with 33 (the SIGINT was forwarded to the container), and the
	// CLI process waited for the container exit and properly captured/set the exit code
	assert.Equal(t, c.ProcessState.ExitCode(), 33)
	assert.Equal(t, d.String(), "exit status 33\n")
}
