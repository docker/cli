package container

import (
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
	result := icmd.RunCmd(shell(t,
		"docker run -d -i --rm %s sh -c 'read; exit %d'", fixtures.AlpineImage, exitcode))
	result.Assert(t, icmd.Success)
	return strings.TrimSpace(result.Stdout())
}

func withStdinNewline(cmd *icmd.Cmd) {
	cmd.Stdin = strings.NewReader("\n")
}
