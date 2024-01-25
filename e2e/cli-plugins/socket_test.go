package cliplugins

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"gotest.tools/v3/assert"
)

// TestPluginSocketBackwardsCompatible executes a plugin binary
// that does not connect to the CLI plugin socket, simulating
// a plugin compiled against an older version of the CLI, and
// ensures that backwards compatibility is maintained.
func TestPluginSocketBackwardsCompatible(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	t.Run("attached", func(t *testing.T) {
		t.Run("the plugin gets signalled if attached to a TTY", func(t *testing.T) {
			cmd := run("presocket", "test-no-socket")
			command := exec.Command(cmd.Command[0], cmd.Command[1:]...)

			ptmx, err := pty.Start(command)
			assert.NilError(t, err, "failed to launch command with fake TTY")

			// send a SIGINT to the process group after 1 second, since
			// we're simulating an "attached TTY" scenario and a TTY would
			// send a signal to the process group
			go func() {
				<-time.After(time.Second)
				err := syscall.Kill(-command.Process.Pid, syscall.SIGINT)
				assert.NilError(t, err, "failed to signal process group")
			}()
			bytes, err := io.ReadAll(ptmx)
			if err != nil && !strings.Contains(err.Error(), "input/output error") {
				t.Fatal("failed to get command output")
			}

			// the plugin is attached to the TTY, so the parent process
			// ignores the received signal, and the plugin receives a SIGINT
			// as well
			assert.Equal(t, string(bytes), "received SIGINT\r\nexit after 3 seconds\r\n")
		})

		// ensure that we don't break plugins that attempt to read from the TTY
		// (see: https://github.com/moby/moby/issues/47073)
		// (remove me if/when we decide to break compatibility here)
		t.Run("the plugin can read from the TTY", func(t *testing.T) {
			cmd := run("presocket", "tty")
			command := exec.Command(cmd.Command[0], cmd.Command[1:]...)

			ptmx, err := pty.Start(command)
			assert.NilError(t, err, "failed to launch command with fake TTY")
			_, _ = ptmx.Write([]byte("hello!"))

			done := make(chan error)
			go func() {
				<-time.After(time.Second)
				_, err := io.ReadAll(ptmx)
				done <- err
			}()

			select {
			case cmdErr := <-done:
				if cmdErr != nil && !strings.Contains(cmdErr.Error(), "input/output error") {
					t.Fatal("failed to get command output")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("timed out! plugin process probably stuck")
			}
		})
	})

	t.Run("detached", func(t *testing.T) {
		t.Run("the plugin does not get signalled", func(t *testing.T) {
			cmd := run("presocket", "test-no-socket")
			command := exec.Command(cmd.Command[0], cmd.Command[1:]...)
			t.Log(strings.Join(command.Args, " "))
			command.SysProcAttr = &syscall.SysProcAttr{
				Setpgid: true,
			}

			go func() {
				<-time.After(time.Second)
				// we're signalling the parent process directly and not
				// the process group, since we're testing the case where
				// the process is detached and not simulating a CTRL-C
				// from a TTY
				err := syscall.Kill(command.Process.Pid, syscall.SIGINT)
				assert.NilError(t, err, "failed to signal process group")
			}()
			bytes, err := command.CombinedOutput()
			t.Log("command output: " + string(bytes))
			assert.NilError(t, err, "failed to run command")

			// the plugin process does not receive a SIGINT
			// so it exits after 3 seconds and prints this message
			assert.Equal(t, string(bytes), "exit after 3 seconds\n")
		})

		t.Run("the main CLI exits after 3 signals", func(t *testing.T) {
			cmd := run("presocket", "test-no-socket")
			command := exec.Command(cmd.Command[0], cmd.Command[1:]...)
			t.Log(strings.Join(command.Args, " "))
			command.SysProcAttr = &syscall.SysProcAttr{
				Setpgid: true,
			}

			go func() {
				<-time.After(time.Second)
				// we're signalling the parent process directly and not
				// the process group, since we're testing the case where
				// the process is detached and not simulating a CTRL-C
				// from a TTY
				err := syscall.Kill(command.Process.Pid, syscall.SIGINT)
				assert.NilError(t, err, "failed to signal process group")
				// TODO: look into CLI signal handling, it's currently necessary
				// to add a short delay between each signal in order for the CLI
				// process to consistently pick them all up.
				time.Sleep(50 * time.Millisecond)
				err = syscall.Kill(command.Process.Pid, syscall.SIGINT)
				assert.NilError(t, err, "failed to signal process group")
				time.Sleep(50 * time.Millisecond)
				err = syscall.Kill(command.Process.Pid, syscall.SIGINT)
				assert.NilError(t, err, "failed to signal process group")
			}()
			bytes, err := command.CombinedOutput()
			assert.ErrorContains(t, err, "exit status 1")

			// the plugin process does not receive a SIGINT and does
			// the CLI cannot cancel it over the socket, so it kills
			// the plugin process and forcefully exits
			assert.Equal(t, string(bytes), "got 3 SIGTERM/SIGINTs, forcefully exiting\n")
		})
	})
}

func TestPluginSocketCommunication(t *testing.T) {
	run, _, cleanup := prepare(t)
	defer cleanup()

	t.Run("attached", func(t *testing.T) {
		t.Run("the socket is not closed + the plugin receives a signal due to pgid", func(t *testing.T) {
			cmd := run("presocket", "test-socket")
			command := exec.Command(cmd.Command[0], cmd.Command[1:]...)

			ptmx, err := pty.Start(command)
			assert.NilError(t, err, "failed to launch command with fake TTY")

			// send a SIGINT to the process group after 1 second, since
			// we're simulating an "attached TTY" scenario and a TTY would
			// send a signal to the process group
			go func() {
				<-time.After(time.Second)
				err := syscall.Kill(-command.Process.Pid, syscall.SIGINT)
				assert.NilError(t, err, "failed to signal process group")
			}()
			bytes, err := io.ReadAll(ptmx)
			if err != nil && !strings.Contains(err.Error(), "input/output error") {
				t.Fatal("failed to get command output")
			}

			// the plugin is attached to the TTY, so the parent process
			// ignores the received signal, and the plugin receives a SIGINT
			// as well
			assert.Equal(t, string(bytes), "received SIGINT\r\nexit after 3 seconds\r\n")
		})
	})

	t.Run("detached", func(t *testing.T) {
		t.Run("the plugin does not get signalled", func(t *testing.T) {
			cmd := run("presocket", "test-socket")
			command := exec.Command(cmd.Command[0], cmd.Command[1:]...)
			outB := bytes.Buffer{}
			command.Stdout = &outB
			command.Stderr = &outB
			command.SysProcAttr = &syscall.SysProcAttr{
				Setpgid: true,
			}

			// send a SIGINT to the process group after 1 second
			go func() {
				<-time.After(time.Second)
				err := syscall.Kill(command.Process.Pid, syscall.SIGINT)
				assert.NilError(t, err, "failed to signal CLI process")
			}()
			err := command.Run()
			t.Log(outB.String())
			assert.ErrorContains(t, err, "exit status 2")

			// the plugin does not get signalled, but it does get it's
			// context cancelled by the CLI through the socket
			assert.Equal(t, outB.String(), "context cancelled\n")
		})

		t.Run("the main CLI exits after 3 signals", func(t *testing.T) {
			cmd := run("presocket", "test-socket-ignore-context")
			command := exec.Command(cmd.Command[0], cmd.Command[1:]...)
			command.SysProcAttr = &syscall.SysProcAttr{
				Setpgid: true,
			}

			go func() {
				<-time.After(time.Second)
				// we're signalling the parent process directly and not
				// the process group, since we're testing the case where
				// the process is detached and not simulating a CTRL-C
				// from a TTY
				err := syscall.Kill(command.Process.Pid, syscall.SIGINT)
				assert.NilError(t, err, "failed to signal CLI process")
				// TODO: same as above TODO, CLI signal handling is not consistent
				// with multiple signals without intervals
				time.Sleep(50 * time.Millisecond)
				err = syscall.Kill(command.Process.Pid, syscall.SIGINT)
				assert.NilError(t, err, "failed to signal CLI process")
				time.Sleep(50 * time.Millisecond)
				err = syscall.Kill(command.Process.Pid, syscall.SIGINT)
				assert.NilError(t, err, "failed to signal CLI process§")
			}()
			bytes, err := command.CombinedOutput()
			assert.ErrorContains(t, err, "exit status 1")

			// the plugin process does not receive a SIGINT and does
			// not exit after having it's context cancelled, so the CLI
			// kills the plugin process and forcefully exits
			assert.Equal(t, string(bytes), "got 3 SIGTERM/SIGINTs, forcefully exiting\n")
		})
	})
}
