package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// RemoveOptions contains options for the docker config rm command.
//
// Deprecated: this type was for internal use and will be removed in the next release.
type RemoveOptions struct {
	Names []string
}

func newConfigRemoveCommand(dockerCLI command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:     "rm CONFIG [CONFIG...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more configs",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), dockerCLI, args)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeNames(dockerCLI)(cmd, args, toComplete)
		},
	}
}

// RunConfigRemove removes the given Swarm configs.
//
// Deprecated: this function was for internal use and will be removed in the next release.
func RunConfigRemove(ctx context.Context, dockerCLI command.Cli, opts RemoveOptions) error {
	return runRemove(ctx, dockerCLI, opts.Names)
}

// runRemove removes the given Swarm configs.
func runRemove(ctx context.Context, dockerCLI command.Cli, names []string) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, name := range names {
		if err := apiClient.ConfigRemove(ctx, name); err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	}

	return errors.Join(errs...)
}
