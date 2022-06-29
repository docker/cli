package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/cli/cli/command/completion"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type killOptions struct {
	all    bool
	signal string

	containers []string
}

// NewKillCommand creates a new cobra.Command for `docker kill`
func NewKillCommand(dockerCli command.Cli) *cobra.Command {
	var opts killOptions

	cmd := &cobra.Command{
		Use:   "kill [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Kill one or more running containers",
		Args:  cli.RequiresArgOrAllFlag(),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runKill(dockerCli, &opts)
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, false),
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.all, "all", false, "Kill all containers")
	flags.StringVarP(&opts.signal, "signal", "s", "KILL", "Signal to send to the container")
	return cmd
}

func runKill(dockerCli command.Cli, opts *killOptions) error {
	var errs []string
	ctx := context.Background()

	if opts.all {
		filter := filters.NewArgs(filters.KeyValuePair{Key: "status", Value: "running"})
		containers, err := dockerCli.Client().ContainerList(ctx, types.ContainerListOptions{Filters: filter})
		if err != nil {
			return err
		}
		if len(containers) == 0 {
			return fmt.Errorf("no containers running to send %s signal to", opts.signal)
		}
		for _, container := range containers {
			opts.containers = append(opts.containers, container.ID)
		}
	}

	errChan := parallelOperation(ctx, opts.containers, func(ctx context.Context, container string) error {
		return dockerCli.Client().ContainerKill(ctx, container, opts.signal)
	})
	for _, name := range opts.containers {
		if err := <-errChan; err != nil {
			errs = append(errs, err.Error())
		} else {
			fmt.Fprintln(dockerCli.Out(), name)
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
