package socket

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"os"
	"time"
)

// EnvKey represents the well-known environment variable used to pass the plugin being
// executed the socket name it should listen on to coordinate with the host CLI.
const EnvKey = "DOCKER_CLI_PLUGIN_SOCKET"

// SetupConn sets up a Unix socket listener, establishes a goroutine to handle connections
// and update the conn pointer, and returns the listener for the socket (which the caller
// is responsible for closing when it's no longer needed).
func SetupConn() (*net.UnixListener, <-chan *net.UnixConn, error) {
	listener, err := listen("docker_cli_" + randomID())
	if err != nil {
		return nil, nil, err
	}

	// accept starts a background goroutine
	// to accept a new connection
	// once accepted, the connChan will be updated.
	connChan := accept(listener)

	return listener, connChan, nil
}

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // This shouldn't happen
	}
	return hex.EncodeToString(b)
}

// accept creates a new Unix socket connection
// and sends it to the *net.UnixConn channel
// it allows reconnects
func accept(listener *net.UnixListener) <-chan *net.UnixConn {
	connChan := make(chan *net.UnixConn, 1)

	go func() {
		const maxRetries = 10
		const waitBetweenRetries = 100 * time.Millisecond

		var conn *net.UnixConn
		var err error

		// retry accepting a connection if there was an error
		for i := 0; i < maxRetries; i++ {
			// this is a blocking call and will wait
			// until a new connection is accepted
			// or until the timout is reached
			conn, err = listener.AcceptUnix()

			if err != nil {
				time.Sleep(waitBetweenRetries)
				continue
			}
			break
		}
		// perform any platform-specific actions on accept (e.g. unlink non-abstract sockets)
		onAccept(listener)
		connChan <- conn
		// close the channel to signal we won't accept any more connections
		close(connChan)
	}()

	return connChan
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
