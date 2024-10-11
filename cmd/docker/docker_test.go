package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/debug"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/cmd/docker/internal/signals"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/poll"
)

func TestClientDebugEnabled(t *testing.T) {
	defer debug.Disable()
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cli, err := command.NewDockerCli(command.WithBaseContext(ctx))
	assert.NilError(t, err)
	tcmd := newDockerCommand(cli)
	tcmd.SetFlag("debug", "true")
	cmd, _, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)
	assert.NilError(t, tcmd.Initialize())
	err = cmd.PersistentPreRunE(cmd, []string{})
	assert.NilError(t, err)
	assert.Check(t, is.Equal("1", os.Getenv("DEBUG")))
	assert.Check(t, is.Equal(logrus.DebugLevel, logrus.GetLevel()))
}

var discard = io.NopCloser(bytes.NewBuffer(nil))

func runCliCommand(t *testing.T, r io.ReadCloser, w io.Writer, args ...string) error {
	t.Helper()
	if r == nil {
		r = discard
	}
	if w == nil {
		w = io.Discard
	}
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cli, err := command.NewDockerCli(
		command.WithBaseContext(ctx),
		command.WithInputStream(r),
		command.WithCombinedStreams(w))
	assert.NilError(t, err)
	tcmd := newDockerCommand(cli)

	tcmd.SetArgs(args)
	cmd, _, err := tcmd.HandleGlobalFlags()
	assert.NilError(t, err)
	assert.NilError(t, tcmd.Initialize())
	return cmd.Execute()
}

func TestExitStatusForInvalidSubcommandWithHelpFlag(t *testing.T) {
	err := runCliCommand(t, nil, nil, "help", "invalid")
	assert.Error(t, err, "unknown help topic: invalid")
}

func TestExitStatusForInvalidSubcommand(t *testing.T) {
	err := runCliCommand(t, nil, nil, "invalid")
	assert.Check(t, is.ErrorContains(err, "docker: unknown command: docker invalid"))
}

func TestVersion(t *testing.T) {
	var b bytes.Buffer
	err := runCliCommand(t, nil, &b, "--version")
	assert.NilError(t, err)
	assert.Check(t, is.Contains(b.String(), "Docker version"))
}

func TestFallbackForceExit(t *testing.T) {
	longRunningCommand := cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			read, _, err := os.Pipe()
			if err != nil {
				return err
			}

			// wait until the parent process sends a signal to exit
			_, _, err = bufio.NewReader(read).ReadLine()
			return err
		},
	}

	// This is the child process that will run the long running command
	if os.Getenv("TEST_FALLBACK_FORCE_EXIT") == "1" {
		fmt.Println("running long command")
		ctx, cancel := signal.NotifyContext(context.Background(), signals.TerminationSignals...)
		t.Cleanup(cancel)

		longRunningCommand.SetErr(streams.NewOut(os.Stderr))
		longRunningCommand.SetOut(streams.NewOut(os.Stdout))

		go forceExitAfter3TerminationSignals(ctx, streams.NewOut(os.Stderr))

		err := longRunningCommand.ExecuteContext(ctx)
		if err != nil {
			os.Exit(0)
		}
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// spawn the child process
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestFallbackForceExit")
	cmd.Env = append(os.Environ(), "TEST_FALLBACK_FORCE_EXIT=1")

	var buf strings.Builder
	cmd.Stderr = &buf
	cmd.Stdout = &buf

	t.Cleanup(func() {
		_ = cmd.Process.Kill()
	})

	assert.NilError(t, cmd.Start())

	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		if strings.Contains(buf.String(), "running long command") {
			return poll.Success()
		}
		return poll.Continue("waiting for child process to start")
	}, poll.WithTimeout(1*time.Second), poll.WithDelay(100*time.Millisecond))

	for i := 0; i < 3; i++ {
		cmd.Process.Signal(syscall.SIGINT)
		time.Sleep(100 * time.Millisecond)
	}

	cmdErr := make(chan error, 1)
	go func() {
		cmdErr <- cmd.Wait()
	}()

	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		if strings.Contains(buf.String(), "got 3 SIGTERM/SIGINTs, forcefully exiting") {
			return poll.Success()
		}
		return poll.Continue("waiting for child process to exit")
	},
		poll.WithTimeout(1*time.Second), poll.WithDelay(100*time.Millisecond))

	select {
	case cmdErr := <-cmdErr:
		assert.Error(t, cmdErr, "exit status 1")
		exitErr, ok := cmdErr.(*exec.ExitError)
		if !ok {
			t.Fatalf("unexpected error type: %T", cmdErr)
		}
		if exitErr.Success() {
			t.Fatalf("unexpected exit status: %v", exitErr)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for child process to exit")
	}
}
