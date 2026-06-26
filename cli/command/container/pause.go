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

type pauseOptions struct {
	containers []string
}

// newPauseCommand creates a new cobra.Command for "docker container pause"
func newPauseCommand(dockerCLI command.Cli) *cobra.Command {
	var opts pauseOptions

	return &cobra.Command{
		Use:   "pause CONTAINER [CONTAINER...]",
		Short: "Pause all processes within one or more containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runPause(cmd.Context(), dockerCLI, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container pause, docker pause",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCLI, false, func(ctr container.Summary) bool {
			return ctr.State != container.StatePaused
		}),
		DisableFlagsInUseLine: true,
	}
}

func runPause(ctx context.Context, dockerCLI command.Cli, opts *pauseOptions) error {
	apiClient := dockerCLI.Client()
	errChan := parallelOperation(ctx, opts.containers, func(ctx context.Context, container string) error {
		_, err := apiClient.ContainerPause(ctx, container, client.ContainerPauseOptions{})
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
