//go:build !windows

package commandconn

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// For https://github.com/docker/cli/pull/1014#issuecomment-409308139
func TestEOFWithError(t *testing.T) {
	ctx := context.TODO()
	c, err := New(ctx, "sh", "-c", "echo hello; echo some error >&2; exit 42")
	assert.NilError(t, err)
	b := make([]byte, 32)
	n, err := c.Read(b)
	assert.Check(t, is.Equal(len("hello\n"), n))
	assert.NilError(t, err)
	n, err = c.Read(b)
	assert.Check(t, is.Equal(0, n))
	assert.ErrorContains(t, err, "some error")
	assert.ErrorContains(t, err, "42")
}

func TestEOFWithoutError(t *testing.T) {
	ctx := context.TODO()
	c, err := New(ctx, "sh", "-c", "echo hello; echo some debug log >&2; exit 0")
	assert.NilError(t, err)
	b := make([]byte, 32)
	n, err := c.Read(b)
	assert.Check(t, is.Equal(len("hello\n"), n))
	assert.NilError(t, err)
	n, err = c.Read(b)
	assert.Check(t, is.Equal(0, n))
	assert.Check(t, is.Equal(io.EOF, err))
}

func TestCloseRunningCommand(t *testing.T) {
	ctx := context.TODO()
	done := make(chan struct{})
	defer close(done)

	go func() {
		c, err := New(ctx, "sh", "-c", "while true; do sleep 1; done")
		assert.NilError(t, err)
		cmdConn := c.(*commandConn)
		assert.Check(t, processAlive(cmdConn.cmd.Process.Pid))

		n, err := c.Write([]byte("hello"))
		assert.Check(t, is.Equal(len("hello"), n))
		assert.NilError(t, err)
		assert.Check(t, processAlive(cmdConn.cmd.Process.Pid))

		err = cmdConn.Close()
		assert.NilError(t, err)
		assert.Check(t, !processAlive(cmdConn.cmd.Process.Pid))
		done <- struct{}{}
	}()

	select {
	case <-time.After(5 * time.Second):
		t.Error("test did not finish in time")
	case <-done:
		break
	}
}

func TestCloseTwice(t *testing.T) {
	ctx := context.TODO()
	done := make(chan struct{})
	go func() {
		c, err := New(ctx, "sh", "-c", "echo hello; sleep 1; exit 0")
		assert.NilError(t, err)
		cmdConn := c.(*commandConn)
		assert.Check(t, processAlive(cmdConn.cmd.Process.Pid))

		b := make([]byte, 32)
		n, err := c.Read(b)
		assert.Check(t, is.Equal(len("hello\n"), n))
		assert.NilError(t, err)

		err = cmdConn.Close()
		assert.NilError(t, err)
		assert.Check(t, !processAlive(cmdConn.cmd.Process.Pid))

		err = cmdConn.Close()
		assert.NilError(t, err)
		assert.Check(t, !processAlive(cmdConn.cmd.Process.Pid))
		done <- struct{}{}
	}()

	select {
	case <-time.After(10 * time.Second):
		t.Error("test did not finish in time")
	case <-done:
		break
	}
}

func TestEOFTimeout(t *testing.T) {
	ctx := context.TODO()
	done := make(chan struct{})
	go func() {
		c, err := New(ctx, "sh", "-c", "sleep 20")
		assert.NilError(t, err)
		cmdConn := c.(*commandConn)
		assert.Check(t, processAlive(cmdConn.cmd.Process.Pid))

		cmdConn.stdout = mockStdoutEOF{}

		b := make([]byte, 32)
		n, err := c.Read(b)
		assert.Check(t, is.Equal(0, n))
		assert.ErrorContains(t, err, "did not exit after EOF")

		done <- struct{}{}
	}()

	// after receiving an EOF, we try to kill the command
	// if it doesn't exit after 10s, we throw an error
	select {
	case <-time.After(12 * time.Second):
		t.Error("test did not finish in time")
	case <-done:
		break
	}
}

type mockStdoutEOF struct{}

func (mockStdoutEOF) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (mockStdoutEOF) Close() error {
	return nil
}

func TestCloseWhileWriting(t *testing.T) {
	ctx := context.TODO()
	c, err := New(ctx, "sh", "-c", "while true; do sleep 1; done")
	assert.NilError(t, err)
	cmdConn := c.(*commandConn)
	assert.Check(t, processAlive(cmdConn.cmd.Process.Pid))

	writeErrC := make(chan error)
	go func() {
		for {
			n, err := c.Write([]byte("hello"))
			if err != nil {
				writeErrC <- err
				return
			}
			assert.Equal(t, n, len("hello"))
		}
	}()

	err = c.Close()
	assert.NilError(t, err)
	assert.Check(t, !processAlive(cmdConn.cmd.Process.Pid))

	writeErr := <-writeErrC
	assert.ErrorContains(t, writeErr, "file already closed")
	assert.Check(t, is.ErrorIs(writeErr, fs.ErrClosed))
}

func TestCloseWhileReading(t *testing.T) {
	ctx := context.TODO()
	c, err := New(ctx, "sh", "-c", "while true; do sleep 1; done")
	assert.NilError(t, err)
	cmdConn := c.(*commandConn)
	assert.Check(t, processAlive(cmdConn.cmd.Process.Pid))

	readErrC := make(chan error)
	go func() {
		for {
			b := make([]byte, 32)
			n, err := c.Read(b)
			if err != nil {
				readErrC <- err
				return
			}
			assert.Check(t, is.Equal(0, n))
		}
	}()

	err = cmdConn.Close()
	assert.NilError(t, err)
	assert.Check(t, !processAlive(cmdConn.cmd.Process.Pid))

	readErr := <-readErrC
	assert.Check(t, is.ErrorIs(readErr, fs.ErrClosed))
}

// processAlive returns true if a process with a given pid is running. It only considers
// positive PIDs; 0 (all processes in the current process group), -1 (all processes
// with a PID larger than 1), and negative (-n, all processes in process group
// "n") values for pid are never considered to be alive.
//
// It was forked from https://github.com/moby/moby/blob/v28.3.3/pkg/process/process_unix.go#L17-L42
func processAlive(pid int) bool {
	if pid < 1 {
		return false
	}
	switch runtime.GOOS {
	case "darwin":
		// OS X does not have a proc filesystem. Use kill -0 pid to judge if the
		// process exists. From KILL(2): https://www.freebsd.org/cgi/man.cgi?query=kill&sektion=2&manpath=OpenDarwin+7.2.1
		//
		// Sig may be one of the signals specified in sigaction(2) or it may
		// be 0, in which case error checking is performed but no signal is
		// actually sent. This can be used to check the validity of pid.
		err := syscall.Kill(pid, 0)

		// Either the PID was found (no error), or we get an EPERM, which means
		// the PID exists, but we don't have permissions to signal it.
		return err == nil || errors.Is(err, syscall.EPERM)
	default:
		_, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
		return err == nil
	}
}
