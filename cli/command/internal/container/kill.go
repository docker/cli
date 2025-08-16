package container

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/command/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.RegisterCommand(newKillCommand)
}

type killOptions struct {
	signal string

	containers []string
}

// NewKillCommand creates a new cobra.Command for `docker kill`
//
// This is a legacy command that can be hidden by setting the `DOCKER_HIDE_LEGACY_COMMANDS`
// environment variable.
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewKillCommand(dockerCli command.Cli) *cobra.Command {
	return newKillCommand(dockerCli)
}

// newKillCommand creates a new cobra.Command for `docker kill`
func newKillCommand(dockerCli command.Cli) *cobra.Command {
	var opts killOptions

	cmd := &cobra.Command{
		Use:   "kill [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Kill one or more running containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runKill(cmd.Context(), dockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container kill, docker kill",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, false),
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.signal, "signal", "s", "", "Signal to send to the container")

	_ = cmd.RegisterFlagCompletionFunc("signal", completeSignals)

	return cmd
}

func runKill(ctx context.Context, dockerCLI command.Cli, opts *killOptions) error {
	apiClient := dockerCLI.Client()
	errChan := parallelOperation(ctx, opts.containers, func(ctx context.Context, container string) error {
		return apiClient.ContainerKill(ctx, container, opts.signal)
	})

	var errs []error
	for _, name := range opts.containers {
		if err := <-errChan; err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	}
	return errors.Join(errs...)
}
