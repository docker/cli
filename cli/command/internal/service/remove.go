package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/spf13/cobra"
)

func newRemoveCommand(dockerCLI cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm SERVICE [SERVICE...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more services",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), dockerCLI, args)
		},
		ValidArgsFunction: completeServiceNames(dockerCLI),
	}
	cmd.Flags()

	return cmd
}

func runRemove(ctx context.Context, dockerCLI cli.Cli, serviceIDs []string) error {
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
