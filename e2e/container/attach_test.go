package container

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/icmd"
)

func TestAttachExitCode(t *testing.T) {
	cName := "test-attach-exit-code"
	icmd.RunCommand("docker", "run", "-d", "--rm", "--name", cName,
		alpineImage, "sh", "-c", "sleep 5 ; exit 21").Assert(t, icmd.Success)
	cmd := icmd.Command("docker", "wait", cName)
	res := icmd.StartCmd(cmd)
	icmd.RunCommand("docker", "attach", cName).Assert(t, icmd.Expected{ExitCode: 21})
	icmd.WaitOnCmd(8, res).Assert(t, icmd.Expected{ExitCode: 0, Out: "21"})
}
