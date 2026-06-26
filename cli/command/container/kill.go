package container

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type killOptions struct {
	signal string

	containers []string
}

// newKillCommand creates a new cobra.Command for "docker container kill"
func newKillCommand(dockerCLI command.Cli) *cobra.Command {
	var opts killOptions

	cmd := &cobra.Command{
		Use:   "kill [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Kill one or more running containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runKill(cmd.Context(), dockerCLI, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container kill, docker kill",
		},
		ValidArgsFunction:     completion.ContainerNames(dockerCLI, false),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.signal, "signal", "s", "", "Signal to send to the container")

	_ = cmd.RegisterFlagCompletionFunc("signal", completeSignals)

	return cmd
}

func runKill(ctx context.Context, dockerCLI command.Cli, opts *killOptions) error {
	apiClient := dockerCLI.Client()
	errChan := parallelOperation(ctx, opts.containers, func(ctx context.Context, container string) error {
		_, err := apiClient.ContainerKill(ctx, container, client.ContainerKillOptions{
			Signal: opts.signal,
		})
		return err
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
