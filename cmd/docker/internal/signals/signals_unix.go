//go:build !windows

package signals

import (
	"os"

	"golang.org/x/sys/unix"
)

// TerminationSignals represents the list of signals we
// want to special-case handle, on this platform.
var TerminationSignals = []os.Signal{unix.SIGTERM, unix.SIGINT}
