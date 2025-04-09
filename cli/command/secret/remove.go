package secret

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	names []string
}

func newSecretRemoveCommand(dockerCLI command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:     "rm SECRET [SECRET...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more secrets",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := removeOptions{
				names: args,
			}
			return runRemove(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeNames(dockerCLI)(cmd, args, toComplete)
		},
	}
}

func runRemove(ctx context.Context, dockerCLI command.Cli, opts removeOptions) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, name := range opts.names {
		if err := apiClient.SecretRemove(ctx, name); err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	}
	return errors.Join(errs...)
}
