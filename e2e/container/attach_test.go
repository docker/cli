package container

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/cli/e2e/internal/fixtures"
	"github.com/gotestyourself/gotestyourself/icmd"
)

func TestAttachExitCode(t *testing.T) {
	containerID := runBackgroundContainsWithExitCode(t, 21)

	result := icmd.RunCmd(
		icmd.Command("docker", "attach", containerID),
		withStdinNewline)

	result.Assert(t, icmd.Expected{ExitCode: 21})
}

func runBackgroundContainsWithExitCode(t *testing.T, exitcode int) string {
	result := icmd.RunCommand("docker", "run", "-d", "-i", "--rm", fixtures.AlpineImage,
		"sh", "-c", fmt.Sprintf("read; exit %d", exitcode))
	result.Assert(t, icmd.Success)
	return strings.TrimSpace(result.Stdout())
}

func withStdinNewline(cmd *icmd.Cmd) {
	cmd.Stdin = strings.NewReader("\n")
}
