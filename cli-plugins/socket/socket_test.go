package socket

import (
	"io/fs"
	"net"
	"os"
	"runtime"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/poll"
)

func TestSetupConn(t *testing.T) {
	t.Run("updates conn when connected", func(t *testing.T) {
		var conn *net.UnixConn
		listener, err := SetupConn(&conn)
		assert.NilError(t, err)
		assert.Check(t, listener != nil, "returned nil listener but no error")
		addr, err := net.ResolveUnixAddr("unix", listener.Addr().String())
		assert.NilError(t, err, "failed to resolve listener address")

		_, err = net.DialUnix("unix", nil, addr)
		assert.NilError(t, err, "failed to dial returned listener")

		pollConnNotNil(t, &conn)
	})

	t.Run("allows reconnects", func(t *testing.T) {
		var conn *net.UnixConn
		listener, err := SetupConn(&conn)
		assert.NilError(t, err)
		assert.Check(t, listener != nil, "returned nil listener but no error")
		addr, err := net.ResolveUnixAddr("unix", listener.Addr().String())
		assert.NilError(t, err, "failed to resolve listener address")

		otherConn, err := net.DialUnix("unix", nil, addr)
		assert.NilError(t, err, "failed to dial returned listener")

		otherConn.Close()

		_, err = net.DialUnix("unix", nil, addr)
		assert.NilError(t, err, "failed to redial listener")
	})

	t.Run("does not leak sockets to local directory", func(t *testing.T) {
		var conn *net.UnixConn
		listener, err := SetupConn(&conn)
		assert.NilError(t, err)
		assert.Check(t, listener != nil, "returned nil listener but no error")
		checkDirClean(t)

		addr, err := net.ResolveUnixAddr("unix", listener.Addr().String())
		assert.NilError(t, err, "failed to resolve listener address")
		_, err = net.DialUnix("unix", nil, addr)
		assert.NilError(t, err, "failed to dial returned listener")
		checkDirClean(t)
	})
}

func checkDirClean(t *testing.T) {
	t.Helper()

	files, err := os.ReadDir(".")
	assert.NilError(t, err, "failed to list files in dir to check for leaked sockets")

	for _, f := range files {
		info, err := f.Info()
		assert.NilError(t, err, "failed to check file info")
		if info.Mode().Type() == fs.ModeSocket {
			t.Fatal("found socket in a local directory")
		}
	}
}

func TestConnectAndWait(t *testing.T) {
	t.Run("calls cancel func on EOF", func(t *testing.T) {
		var conn *net.UnixConn
		listener, err := SetupConn(&conn)
		assert.NilError(t, err, "failed to setup listener")

		done := make(chan struct{})
		t.Setenv(EnvKey, listener.Addr().String())
		cancelFunc := func() {
			done <- struct{}{}
		}
		ConnectAndWait(cancelFunc)
		pollConnNotNil(t, &conn)
		conn.Close()

		select {
		case <-done:
		case <-time.After(10 * time.Millisecond):
			t.Fatal("cancel function not closed after 10ms")
		}
	})

	t.Run("connect goroutine exits after EOF", func(t *testing.T) {
		var conn *net.UnixConn
		listener, err := SetupConn(&conn)
		assert.NilError(t, err, "failed to setup listener")
		t.Setenv(EnvKey, listener.Addr().String())
		numGoroutines := runtime.NumGoroutine()

		ConnectAndWait(func() {})
		assert.Equal(t, runtime.NumGoroutine(), numGoroutines+1)

		pollConnNotNil(t, &conn)
		conn.Close()
		poll.WaitOn(t, func(t poll.LogT) poll.Result {
			if runtime.NumGoroutine() > numGoroutines+1 {
				return poll.Continue("waiting for connect goroutine to exit")
			}
			return poll.Success()
		}, poll.WithDelay(1*time.Millisecond), poll.WithTimeout(10*time.Millisecond))
	})
}

func pollConnNotNil(t *testing.T, conn **net.UnixConn) {
	t.Helper()

	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		if *conn == nil {
			return poll.Continue("waiting for conn to not be nil")
		}
		return poll.Success()
	}, poll.WithDelay(1*time.Millisecond), poll.WithTimeout(10*time.Millisecond))
}
