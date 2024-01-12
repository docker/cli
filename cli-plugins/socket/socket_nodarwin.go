//go:build !darwin

package socket

import (
	"net"
)

func listen(socketname string) (*net.UnixListener, error) {
	return net.ListenUnix("unix", &net.UnixAddr{
		Name: "@" + socketname,
		Net:  "unix",
	})
}

func onAccept(conn *net.UnixConn, listener *net.UnixListener) {
	// do nothing
	// while on darwin we would unlink here; on non-darwin the socket is abstract and not present on the filesystem
}
