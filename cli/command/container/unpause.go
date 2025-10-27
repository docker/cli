package container

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type unpauseOptions struct {
	containers []string
}

// newUnpauseCommand creates a new cobra.Command for "docker container unpause".
func newUnpauseCommand(dockerCLI command.Cli) *cobra.Command {
	var opts unpauseOptions

	cmd := &cobra.Command{
		Use:   "unpause CONTAINER [CONTAINER...]",
		Short: "Unpause all processes within one or more containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runUnpause(cmd.Context(), dockerCLI, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container unpause, docker unpause",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCLI, false, func(ctr container.Summary) bool {
			return ctr.State == container.StatePaused
		}),
		DisableFlagsInUseLine: true,
	}
	return cmd
}

func runUnpause(ctx context.Context, dockerCLI command.Cli, opts *unpauseOptions) error {
	apiClient := dockerCLI.Client()
	errChan := parallelOperation(ctx, opts.containers, func(ctx context.Context, container string) error {
		_, err := apiClient.ContainerUnpause(ctx, container, client.ContainerUnPauseOptions{})
		return err
	})
	var errs []error
	for _, ctr := range opts.containers {
		if err := <-errChan; err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), ctr)
	}
	return errors.Join(errs...)
}
