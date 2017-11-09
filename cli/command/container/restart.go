package container

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type restartOptions struct {
	nSeconds        int
	nSecondsChanged bool

	nAll bool

	containers []string
}

// NewRestartCommand creates a new cobra.Command for `docker restart`
func NewRestartCommand(dockerCli command.Cli) *cobra.Command {
	var opts restartOptions

	cmd := &cobra.Command{
		Use:   "restart [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Restart one or more containers",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 && !cmd.Flags().Changed("all") {
				return errors.New("\"docker restart\" requires at least 1 argument.")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			opts.nSecondsChanged = cmd.Flags().Changed("time")
			return runRestart(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.IntVarP(&opts.nSeconds, "time", "t", 10, "Seconds to wait for stop before killing the container")
	flags.BoolVar(&opts.nAll, "all", false, "Restart all running containers")
	return cmd
}

func runRestart(dockerCli command.Cli, opts *restartOptions) error {
	ctx := context.Background()
	var errs []string
	var timeout *time.Duration
	if opts.nSecondsChanged {
		timeoutValue := time.Duration(opts.nSeconds) * time.Second
		timeout = &timeoutValue
	}

	if opts.nAll {
		containers, err := dockerCli.Client().ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			errs = append(errs, err.Error())
		} else {
			for _, container := range containers {
				opts.containers = append(opts.containers, container.ID)
			}
		}
	}

	for _, name := range opts.containers {
		if err := dockerCli.Client().ContainerRestart(ctx, name, timeout); err != nil {
			errs = append(errs, err.Error())
			continue
		}
		fmt.Fprintln(dockerCli.Out(), name)
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
