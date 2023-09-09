package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type waitOptions struct {
	containers []string
}

// NewWaitCommand creates a new cobra.Command for `docker wait`
func NewWaitCommand(dockerCli command.Cli) *cobra.Command {
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

func runWait(ctx context.Context, dockerCli command.Cli, opts *waitOptions) error {
	var errs []string
	for _, container := range opts.containers {
		resultC, errC := dockerCli.Client().ContainerWait(ctx, container, "")

		select {
		case result := <-resultC:
			fmt.Fprintf(dockerCli.Out(), "%d\n", result.StatusCode)
		case err := <-errC:
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
