//go:build windows || linux

package socket

import (
	"net"
)

func listen(socketname string) (*net.UnixListener, error) {
	// Create an abstract socket -- this socket can be opened by name, but is
	// not present in the filesystem.
	return net.ListenUnix("unix", &net.UnixAddr{
		Name: "@" + socketname,
		Net:  "unix",
	})
}

func unlink(listener *net.UnixListener) {
	// Do nothing; the socket is not present in the filesystem.
}
