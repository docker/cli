package socket

import (
	"errors"
	"io"
	"io/fs"
	"net"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/poll"
)

func TestPluginServer(t *testing.T) {
	t.Run("connection closes with EOF when server closes", func(t *testing.T) {
		called := make(chan struct{})
		srv, err := NewPluginServer(func(_ net.Conn) { close(called) })
		assert.NilError(t, err)
		assert.Assert(t, srv != nil, "returned nil server but no error")

		addr, err := net.ResolveUnixAddr("unix", srv.Addr().String())
		assert.NilError(t, err, "failed to resolve server address")

		conn, err := net.DialUnix("unix", nil, addr)
		assert.NilError(t, err, "failed to dial returned server")
		defer conn.Close()

		done := make(chan error, 1)
		go func() {
			_, err := conn.Read(make([]byte, 1))
			done <- err
		}()

		select {
		case <-called:
		case <-time.After(10 * time.Millisecond):
			t.Fatal("handler not called")
		}

		srv.Close()

		select {
		case err := <-done:
			if !errors.Is(err, io.EOF) {
				t.Fatalf("exepcted EOF error, got: %v", err)
			}
		case <-time.After(10 * time.Millisecond):
		}
	})

	t.Run("allows reconnects", func(t *testing.T) {
		var calls int32
		h := func(_ net.Conn) {
			atomic.AddInt32(&calls, 1)
		}

		srv, err := NewPluginServer(h)
		assert.NilError(t, err)
		defer srv.Close()

		assert.Check(t, srv.Addr() != nil, "returned nil addr but no error")

		addr, err := net.ResolveUnixAddr("unix", srv.Addr().String())
		assert.NilError(t, err, "failed to resolve server address")

		waitForCalls := func(n int) {
			poll.WaitOn(t, func(t poll.LogT) poll.Result {
				if atomic.LoadInt32(&calls) == int32(n) {
					return poll.Success()
				}
				return poll.Continue("waiting for handler to be called")
			})
		}

		otherConn, err := net.DialUnix("unix", nil, addr)
		assert.NilError(t, err, "failed to dial returned server")
		otherConn.Close()
		waitForCalls(1)

		conn, err := net.DialUnix("unix", nil, addr)
		assert.NilError(t, err, "failed to redial server")
		defer conn.Close()
		waitForCalls(2)

		// and again but don't close the existing connection
		conn2, err := net.DialUnix("unix", nil, addr)
		assert.NilError(t, err, "failed to redial server")
		defer conn2.Close()
		waitForCalls(3)

		srv.Close()

		// now make sure we get EOF on the existing connections
		buf := make([]byte, 1)
		_, err = conn.Read(buf)
		assert.ErrorIs(t, err, io.EOF, "expected EOF error, got: %v", err)

		_, err = conn2.Read(buf)
		assert.ErrorIs(t, err, io.EOF, "expected EOF error, got: %v", err)
	})

	t.Run("does not leak sockets to local directory", func(t *testing.T) {
		srv, err := NewPluginServer(nil)
		assert.NilError(t, err)
		assert.Check(t, srv != nil, "returned nil server but no error")
		checkDirNoNewPluginServer(t)

		addr, err := net.ResolveUnixAddr("unix", srv.Addr().String())
		assert.NilError(t, err, "failed to resolve server address")

		_, err = net.DialUnix("unix", nil, addr)
		assert.NilError(t, err, "failed to dial returned server")
		checkDirNoNewPluginServer(t)
	})

	t.Run("does not panic on Close if server is nil", func(t *testing.T) {
		var srv *PluginServer
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panicked on Close")
			}
		}()

		err := srv.Close()
		assert.NilError(t, err)
	})
}

func checkDirNoNewPluginServer(t *testing.T) {
	t.Helper()

	files, err := os.ReadDir(".")
	assert.NilError(t, err, "failed to list files in dir to check for leaked sockets")

	for _, f := range files {
		info, err := f.Info()
		assert.NilError(t, err, "failed to check file info")
		// check for a socket with `docker_cli_` in the name (from `SetupConn()`)
		if strings.Contains(f.Name(), "docker_cli_") && info.Mode().Type() == fs.ModeSocket {
			t.Fatal("found socket in a local directory")
		}
	}
}

func TestConnectAndWait(t *testing.T) {
	t.Run("calls cancel func on EOF", func(t *testing.T) {
		srv, err := NewPluginServer(nil)
		assert.NilError(t, err, "failed to setup server")
		defer srv.Close()

		done := make(chan struct{})
		t.Setenv(EnvKey, srv.Addr().String())
		cancelFunc := func() {
			done <- struct{}{}
		}
		ConnectAndWait(cancelFunc)

		select {
		case <-done:
			t.Fatal("unexpectedly done")
		default:
		}

		srv.Close()

		select {
		case <-done:
		case <-time.After(10 * time.Millisecond):
			t.Fatal("cancel function not closed after 10ms")
		}
	})

	// TODO: this test cannot be executed with `t.Parallel()`, due to
	// relying on goroutine numbers to ensure correct behaviour
	t.Run("connect goroutine exits after EOF", func(t *testing.T) {
		srv, err := NewPluginServer(nil)
		assert.NilError(t, err, "failed to setup server")

		defer srv.Close()

		t.Setenv(EnvKey, srv.Addr().String())
		numGoroutines := runtime.NumGoroutine()

		ConnectAndWait(func() {})
		assert.Equal(t, runtime.NumGoroutine(), numGoroutines+1)

		srv.Close()

		poll.WaitOn(t, func(t poll.LogT) poll.Result {
			if runtime.NumGoroutine() > numGoroutines+1 {
				return poll.Continue("waiting for connect goroutine to exit")
			}
			return poll.Success()
		}, poll.WithDelay(1*time.Millisecond), poll.WithTimeout(10*time.Millisecond))
	})
}
