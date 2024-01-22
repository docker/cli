package socket

import (
	"errors"
	"io"
	"net"
	"os"

	"github.com/docker/distribution/uuid"
)

// EnvKey represents the well-known environment variable used to pass the plugin being
// executed the socket name it should listen on to coordinate with the host CLI.
const EnvKey = "DOCKER_CLI_PLUGIN_SOCKET"

// SetupConn sets up a Unix socket listener, establishes a goroutine to handle connections
// and update the conn pointer, and returns the listener for the socket (which the caller
// is responsible for closing when it's no longer needed).
func SetupConn(conn **net.UnixConn) (*net.UnixListener, error) {
	listener, err := listen("docker_cli_" + uuid.Generate().String())
	if err != nil {
		return nil, err
	}

	accept(listener, conn)

	return listener, nil
}

func accept(listener *net.UnixListener, conn **net.UnixConn) {
	go func() {
		for {
			// ignore error here, if we failed to accept a connection,
			// conn is nil and we fallback to previous behavior
			*conn, _ = listener.AcceptUnix()
			// perform any platform-specific actions on accept (e.g. unlink non-abstract sockets)
			onAccept(*conn, listener)
		}
	}()
}

// ConnectAndWait connects to the socket passed via well-known env var,
// if present, and attempts to read from it until it receives an EOF, at which
// point cb is called.
func ConnectAndWait(cb func()) {
	socketAddr, ok := os.LookupEnv(EnvKey)
	if !ok {
		// if a plugin compiled against a more recent version of docker/cli
		// is executed by an older CLI binary, ignore missing environment
		// variable and behave as usual
		return
	}
	addr, err := net.ResolveUnixAddr("unix", socketAddr)
	if err != nil {
		return
	}
	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return
	}

	go func() {
		b := make([]byte, 1)
		for {
			_, err := conn.Read(b)
			if errors.Is(err, io.EOF) {
				cb()
				return
			}
		}
	}()
}
