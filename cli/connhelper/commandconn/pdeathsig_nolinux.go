//go:build !linux

package commandconn // import "docker.com/cli/v28/cli/connhelper/commandconn"

import (
	"os/exec"
)

func setPdeathsig(*exec.Cmd) {}
