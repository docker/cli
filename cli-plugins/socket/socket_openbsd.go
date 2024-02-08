package socket

import (
	"net"
	"os"
	"path/filepath"
	"syscall"
)

func listen(socketname string) (*net.UnixListener, error) {
	return net.ListenUnix("unix", &net.UnixAddr{
		Name: filepath.Join(os.TempDir(), socketname),
		Net:  "unix",
	})
}

func onAccept(conn *net.UnixConn, listener *net.UnixListener) {
	syscall.Unlink(listener.Addr().String())
}
