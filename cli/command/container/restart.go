package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type restartOptions struct {
	signal         string
	timeout        int
	timeoutChanged bool

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
			opts.timeoutChanged = cmd.Flags().Changed("time")
			return runRestart(cmd.Context(), dockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container restart, docker restart",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, true),
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.signal, "signal", "s", "", "Signal to send to the container")
	flags.IntVarP(&opts.timeout, "time", "t", 0, "Seconds to wait before killing the container")

	_ = cmd.RegisterFlagCompletionFunc("signal", completeSignals)

	return cmd
}

func runRestart(ctx context.Context, dockerCli command.Cli, opts *restartOptions) error {
	var errs []string
	var timeout *int
	if opts.timeoutChanged {
		timeout = &opts.timeout
	}
	for _, name := range opts.containers {
		err := dockerCli.Client().ContainerRestart(ctx, name, container.StopOptions{
			Signal:  opts.signal,
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
