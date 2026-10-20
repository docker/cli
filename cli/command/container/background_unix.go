//go:build !windows

package container

import (
	"golang.org/x/sys/unix"

	"github.com/docker/cli/cli/streams"
)

// stdinFromBackgroundJob reports whether stdin is a TTY whose foreground
// process group is not us. Reading that TTY from a background job (for
// example `docker run -i ... &`) produces SIGTTIN; the CLI catches every
// signal for forwarding, so the default stop-in-background does not fire
// and the read retries in a tight loop.
func stdinFromBackgroundJob(in *streams.In) bool {
	if in == nil || !in.IsTerminal() {
		return false
	}
	fg, err := unix.IoctlGetInt(int(in.FD()), unix.TIOCGPGRP)
	if err != nil {
		return false
	}
	return unix.Getpgrp() != fg
}
