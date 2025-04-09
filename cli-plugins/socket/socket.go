package socket

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"os"
	"runtime"
	"sync"

	"github.com/sirupsen/logrus"
)

// EnvKey represents the well-known environment variable used to pass the
// plugin being executed the socket name it should listen on to coordinate with
// the host CLI.
const EnvKey = "DOCKER_CLI_PLUGIN_SOCKET"

// NewPluginServer creates a plugin server that listens on a new Unix domain
// socket. h is called for each new connection to the socket in a goroutine.
func NewPluginServer(h func(net.Conn)) (*PluginServer, error) {
	// Listen on a Unix socket, with the address being platform-dependent.
	// When a non-abstract address is used, Go will unlink(2) the socket
	// for us once the listener is closed, as documented in
	// [net.UnixListener.SetUnlinkOnClose].
	l, err := net.ListenUnix("unix", &net.UnixAddr{
		Name: socketName("docker_cli_" + randomID()),
		Net:  "unix",
	})
	if err != nil {
		return nil, err
	}
	logrus.Trace("Plugin server listening on ", l.Addr())

	if h == nil {
		h = func(net.Conn) {}
	}

	pl := &PluginServer{
		l: l,
		h: h,
	}

	go func() {
		defer pl.Close()
		for {
			err := pl.accept()
			if err != nil {
				return
			}
		}
	}()

	return pl, nil
}

type PluginServer struct {
	mu     sync.Mutex
	conns  []net.Conn
	l      *net.UnixListener
	h      func(net.Conn)
	closed bool
}

func (pl *PluginServer) accept() error {
	conn, err := pl.l.Accept()
	if err != nil {
		return err
	}

	pl.mu.Lock()
	defer pl.mu.Unlock()

	if pl.closed {
		// Handle potential race between Close and accept.
		conn.Close()
		return errors.New("plugin server is closed")
	}

	pl.conns = append(pl.conns, conn)

	go pl.h(conn)
	return nil
}

// Addr returns the [net.Addr] of the underlying [net.Listener].
func (pl *PluginServer) Addr() net.Addr {
	return pl.l.Addr()
}

// Close ensures that the server is no longer accepting new connections and
// closes all existing connections. Existing connections will receive [io.EOF].
//
// The error value is that of the underlying [net.Listner.Close] call.
func (pl *PluginServer) Close() error {
	if pl == nil {
		return nil
	}
	logrus.Trace("Closing plugin server")
	// Close connections first to ensure the connections get io.EOF instead
	// of a connection reset.
	pl.closeAllConns()

	// Try to ensure that any active connections have a chance to receive
	// io.EOF.
	runtime.Gosched()

	return pl.l.Close()
}

func (pl *PluginServer) closeAllConns() {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	if pl.closed {
		return
	}

	// Prevent new connections from being accepted.
	pl.closed = true

	for _, conn := range pl.conns {
		conn.Close()
	}

	pl.conns = nil
}

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // This shouldn't happen
	}
	return hex.EncodeToString(b)
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
