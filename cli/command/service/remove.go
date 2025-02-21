package service // import "docker.com/cli/v28/cli/command/service"

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/v28/cli"
	"github.com/docker/cli/v28/cli/command"
	"github.com/spf13/cobra"
)

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm SERVICE [SERVICE...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more services",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), dockerCli, args)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return CompletionFn(dockerCli)(cmd, args, toComplete)
		},
	}
	cmd.Flags()

	return cmd
}

func runRemove(ctx context.Context, dockerCLI command.Cli, serviceIDs []string) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, id := range serviceIDs {
		if err := apiClient.ServiceRemove(ctx, id); err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), id)
	}
	return errors.Join(errs...)
}
