//go:build !windows && !linux

package socket

import (
	"net"
	"os"
	"path/filepath"
	"syscall"
)

func listen(socketname string) (*net.UnixListener, error) {
	// Because abstract sockets are unavailable, we create a socket in the
	// system temporary directory instead.
	return net.ListenUnix("unix", &net.UnixAddr{
		Name: filepath.Join(os.TempDir(), socketname),
		Net:  "unix",
	})
}

func unlink(listener *net.UnixListener) {
	// unlink(2) is best effort here; if it fails, we may 'leak' a socket
	// into the filesystem, but this is unlikely and overall harmless.
	_ = syscall.Unlink(listener.Addr().String())
}
