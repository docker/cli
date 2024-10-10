package container

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types/container"

	"github.com/spf13/cobra"
)

type unpauseOptions struct {
	containers []string
}

// NewUnpauseCommand creates a new cobra.Command for `docker unpause`
func NewUnpauseCommand(dockerCli command.Cli) *cobra.Command {
	var opts unpauseOptions

	cmd := &cobra.Command{
		Use:   "unpause CONTAINER [CONTAINER...]",
		Short: "Unpause all processes within one or more containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runUnpause(cmd.Context(), dockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container unpause, docker unpause",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, false, func(ctr container.Summary) bool {
			return ctr.State == "paused"
		}),
	}
	return cmd
}

func runUnpause(ctx context.Context, dockerCli command.Cli, opts *unpauseOptions) error {
	var errs []string
	errChan := parallelOperation(ctx, opts.containers, dockerCli.Client().ContainerUnpause)
	for _, ctr := range opts.containers {
		if err := <-errChan; err != nil {
			errs = append(errs, err.Error())
			continue
		}
		_, _ = fmt.Fprintln(dockerCli.Out(), ctr)
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
