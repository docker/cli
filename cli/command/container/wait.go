package container

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type waitOptions struct {
	containers []string
}

// newWaitCommand creates a new cobra.Command for "docker container wait".
func newWaitCommand(dockerCLI command.Cli) *cobra.Command {
	var opts waitOptions

	cmd := &cobra.Command{
		Use:   "wait CONTAINER [CONTAINER...]",
		Short: "Block until one or more containers stop, then print their exit codes",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runWait(cmd.Context(), dockerCLI, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container wait, docker wait",
		},
		ValidArgsFunction:     completion.ContainerNames(dockerCLI, false),
		DisableFlagsInUseLine: true,
	}

	return cmd
}

func runWait(ctx context.Context, dockerCLI command.Cli, opts *waitOptions) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, ctr := range opts.containers {
		res := apiClient.ContainerWait(ctx, ctr, client.ContainerWaitOptions{})

		select {
		case result := <-res.Result:
			_, _ = fmt.Fprintln(dockerCLI.Out(), strconv.FormatInt(result.StatusCode, 10))
		case err := <-res.Error:
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
