package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/commands"
	"github.com/docker/cli/cli/debug"
	platformsignals "github.com/docker/cli/cmd/docker/internal/signals"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestDisableFlagsInUseLineIsSet(t *testing.T) {
	dockerCli, err := command.NewDockerCli(command.WithBaseContext(context.TODO()))
	assert.NilError(t, err)
	rootCmd := &cobra.Command{DisableFlagsInUseLine: true}
	commands.AddCommands(rootCmd, dockerCli)

	var errs []error
	visitAll(rootCmd, func(c *cobra.Command) {
		if !c.DisableFlagsInUseLine {
			errs = append(errs, errors.New("DisableFlagsInUseLine is not set for "+c.CommandPath()))
		}
	})
	err = errors.Join(errs...)
	assert.NilError(t, err)
}

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

func TestUserTerminatedError(t *testing.T) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), time.Second*1, errors.New("test timeout"))
	t.Cleanup(cancel)

	notifyCtx, cancelNotify := notifyContext(ctx, platformsignals.TerminationSignals...)
	t.Cleanup(cancelNotify)

	assert.Check(t, syscall.Kill(syscall.Getpid(), syscall.SIGINT))

	<-notifyCtx.Done()
	assert.ErrorIs(t, context.Cause(notifyCtx), errCtxSignalTerminated{
		signal: syscall.SIGINT,
	})

	assert.Equal(t, getExitCode(context.Cause(notifyCtx)), 130)

	notifyCtx, cancelNotify = notifyContext(ctx, platformsignals.TerminationSignals...)
	t.Cleanup(cancelNotify)

	assert.Check(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))

	<-notifyCtx.Done()
	assert.ErrorIs(t, context.Cause(notifyCtx), errCtxSignalTerminated{
		signal: syscall.SIGTERM,
	})

	assert.Equal(t, getExitCode(context.Cause(notifyCtx)), 143)
}

func TestVisitAll(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	sub1 := &cobra.Command{Use: "sub1"}
	sub1sub1 := &cobra.Command{Use: "sub1sub1"}
	sub1sub2 := &cobra.Command{Use: "sub1sub2"}
	sub2 := &cobra.Command{Use: "sub2"}

	root.AddCommand(sub1, sub2)
	sub1.AddCommand(sub1sub1, sub1sub2)

	var visited []string
	visitAll(root, func(ccmd *cobra.Command) {
		visited = append(visited, ccmd.Name())
	})
	expected := []string{"sub1sub1", "sub1sub2", "sub1", "sub2", "root"}
	assert.DeepEqual(t, expected, visited)
}
