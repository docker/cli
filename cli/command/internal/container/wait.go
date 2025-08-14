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
	commands.RegisterCommand(newWaitCommand)
}

type waitOptions struct {
	containers []string
}

// NewWaitCommand creates a new cobra.Command for `docker wait`
//
// This is a legacy command that can be hidden by setting the `DOCKER_HIDE_LEGACY_COMMANDS`
// environment variable.
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewWaitCommand(dockerCli command.Cli) *cobra.Command {
	return newWaitCommand(dockerCli)
}

// newWaitCommand creates a new cobra.Command for `docker wait`
func newWaitCommand(dockerCli command.Cli) *cobra.Command {
	var opts waitOptions

	cmd := &cobra.Command{
		Use:   "wait CONTAINER [CONTAINER...]",
		Short: "Block until one or more containers stop, then print their exit codes",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runWait(cmd.Context(), dockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container wait, docker wait",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, false),
	}

	return cmd
}

func runWait(ctx context.Context, dockerCLI command.Cli, opts *waitOptions) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, ctr := range opts.containers {
		resultC, errC := apiClient.ContainerWait(ctx, ctr, "")

		select {
		case result := <-resultC:
			_, _ = fmt.Fprintf(dockerCLI.Out(), "%d\n", result.StatusCode)
		case err := <-errC:
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
