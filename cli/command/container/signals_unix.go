//go:build !windows

package container // import "docker.com/cli/v28/cli/command/container"

import (
	"os"

	"golang.org/x/sys/unix"
)

func isRuntimeSig(s os.Signal) bool {
	return s == unix.SIGURG
}
