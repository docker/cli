//go:build !windows

package manager

import (
	"os/exec"
	"syscall"
)

var defaultSystemPluginDirs = []string{
	"/usr/local/lib/docker/cli-plugins", "/usr/local/libexec/docker/cli-plugins",
	"/usr/lib/docker/cli-plugins", "/usr/libexec/docker/cli-plugins",
}

func configureOSSpecificCommand(cmd *exec.Cmd) {
	// Spawn the plugin process in a new process group, so that signals are not forwarded by the OS.
	// The foreground process group is e.g. sent a SIGINT when Ctrl-C is input to the TTY, but we
	// implement our own job control for the plugin.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
