package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
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

	var errs []string
	for _, id := range serviceIDs {
		err := apiClient.ServiceRemove(ctx, id)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), id)
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
