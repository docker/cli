package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

func newRemoveCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm SERVICE [SERVICE...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more services",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), dockerCLI, args)
		},
		ValidArgsFunction:     completeServiceNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}
	cmd.Flags()

	return cmd
}

func runRemove(ctx context.Context, dockerCLI command.Cli, serviceIDs []string) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, id := range serviceIDs {
		if _, err := apiClient.ServiceRemove(ctx, id, client.ServiceRemoveOptions{}); err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), id)
	}
	return errors.Join(errs...)
}
