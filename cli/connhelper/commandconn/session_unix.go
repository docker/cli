//go:build !windows

package commandconn // import "docker.com/cli/v28/cli/connhelper/commandconn"

import (
	"os/exec"
)

func createSession(cmd *exec.Cmd) {
	// for supporting ssh connection helper with ProxyCommand
	// https://github.com/docker/cli/issues/1707
	cmd.SysProcAttr.Setsid = true
}
