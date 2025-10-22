package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

func newConfigRemoveCommand(dockerCLI command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:     "rm CONFIG [CONFIG...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more configs",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), dockerCLI, args)
		},
		ValidArgsFunction:     completeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}
}

// runRemove removes the given Swarm configs.
func runRemove(ctx context.Context, dockerCLI command.Cli, names []string) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, name := range names {
		if _, err := apiClient.ConfigRemove(ctx, name, client.ConfigRemoveOptions{}); err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	}

	return errors.Join(errs...)
}
