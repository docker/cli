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

type pauseOptions struct {
	containers []string
}

// NewPauseCommand creates a new cobra.Command for `docker pause`
func NewPauseCommand(dockerCli command.Cli) *cobra.Command {
	var opts pauseOptions

	return &cobra.Command{
		Use:   "pause CONTAINER [CONTAINER...]",
		Short: "Pause all processes within one or more containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return runPause(cmd.Context(), dockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container pause, docker pause",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, false, func(ctr container.Summary) bool {
			return ctr.State != "paused"
		}),
	}
}

func runPause(ctx context.Context, dockerCli command.Cli, opts *pauseOptions) error {
	var errs []string
	errChan := parallelOperation(ctx, opts.containers, dockerCli.Client().ContainerPause)
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
