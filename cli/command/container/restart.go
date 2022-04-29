package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type restartOptions struct {
	nSeconds        int
	nSecondsChanged bool

	containers []string
}

// NewRestartCommand creates a new cobra.Command for `docker restart`
func NewRestartCommand(dockerCli command.Cli) *cobra.Command {
	var opts restartOptions

	cmd := &cobra.Command{
		Use:   "restart [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Restart one or more containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			opts.nSecondsChanged = cmd.Flags().Changed("time")
			return runRestart(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.IntVarP(&opts.nSeconds, "time", "t", 10, "Seconds to wait for stop before killing the container")
	return cmd
}

func runRestart(dockerCli command.Cli, opts *restartOptions) error {
	ctx := context.Background()
	var errs []string
	var timeout *int
	if opts.nSecondsChanged {
		timeout = &opts.nSeconds
	}
	for _, name := range opts.containers {
		err := dockerCli.Client().ContainerRestart(ctx, name, container.StopOptions{
			Timeout: timeout,
		})
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		_, _ = fmt.Fprintln(dockerCli.Out(), name)
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
