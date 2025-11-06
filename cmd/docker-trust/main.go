package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli-plugins/metadata"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cmd/docker-trust/internal/version"
	"github.com/docker/cli/cmd/docker-trust/trust"
	"go.opentelemetry.io/otel"
)

func runStandalone(cmd *command.DockerCli) error {
	defer flushMetrics(cmd)
	executable := os.Args[0]
	rootCmd := trust.NewRootCmd(filepath.Base(executable), false, cmd)
	return rootCmd.Execute()
}

// flushMetrics will manually flush metrics from the configured
// meter provider. This is needed when running in standalone mode
// because the meter provider is initialized by the cli library,
// but the mechanism for forcing it to report is not presently
// exposed and not invoked when run in standalone mode.
// There are plans to fix that in the next release, but this is
// needed temporarily until the API for this is more thorough.
func flushMetrics(cmd *command.DockerCli) {
	if mp, ok := cmd.MeterProvider().(command.MeterProvider); ok {
		if err := mp.ForceFlush(context.Background()); err != nil {
			otel.Handle(err)
		}
	}
}

func runPlugin(cmd *command.DockerCli) error {
	rootCmd := trust.NewRootCmd("trust", true, cmd)
	return plugin.RunPlugin(cmd, rootCmd, metadata.Metadata{
		SchemaVersion: "0.1.0",
		Vendor:        "Docker Inc.",
		Version:       version.Version,
	})
}

func run(cmd *command.DockerCli) error {
	if plugin.RunningStandalone() {
		return runStandalone(cmd)
	}
	return runPlugin(cmd)
}

type errCtxSignalTerminated struct {
	signal os.Signal
}

func (errCtxSignalTerminated) Error() string {
	return ""
}

func main() {
	cmd, err := command.NewDockerCli()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err = run(cmd); err == nil {
		return
	}

	if errors.As(err, &errCtxSignalTerminated{}) {
		os.Exit(getExitCode(err))
	}

	if !cerrdefs.IsCanceled(err) {
		if err.Error() != "" {
			_, _ = fmt.Fprintln(cmd.Err(), err)
		}
		os.Exit(getExitCode(err))
	}
}

// getExitCode returns the exit-code to use for the given error.
// If err is a [cli.StatusError] and has a StatusCode set, it uses the
// status-code from it, otherwise it returns "1" for any error.
func getExitCode(err error) int {
	if err == nil {
		return 0
	}

	var userTerminatedErr errCtxSignalTerminated
	if errors.As(err, &userTerminatedErr) {
		s, ok := userTerminatedErr.signal.(syscall.Signal)
		if !ok {
			return 1
		}
		return 128 + int(s)
	}

	var stErr cli.StatusError
	if errors.As(err, &stErr) && stErr.StatusCode != 0 { // FIXME(thaJeztah): StatusCode should never be used with a zero status-code. Check if we do this anywhere.
		return stErr.StatusCode
	}

	// No status-code provided; all errors should have a non-zero exit code.
	return 1
}
