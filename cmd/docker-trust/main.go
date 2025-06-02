package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli-plugins/metadata"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/version"
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

func main() {
	cmd, err := command.NewDockerCli()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err = run(cmd); err == nil {
		return
	}

	// Check the error from the run function above.
	if sterr, ok := err.(cli.StatusError); ok {
		if sterr.Status != "" {
			_, _ = fmt.Fprintln(cmd.Err(), sterr.Status)
		}
		// StatusError should only be used for errors, and all errors should
		// have a non-zero exit status, so never exit with 0
		if sterr.StatusCode == 0 {
			os.Exit(1)
		}
		os.Exit(sterr.StatusCode)
	}

	os.Exit(1)
}
