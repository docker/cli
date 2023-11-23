package container

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"gotest.tools/v3/icmd"
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
