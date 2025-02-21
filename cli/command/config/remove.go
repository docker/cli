package config // import "docker.com/cli/v28/cli/command/config"

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/v28/cli"
	"github.com/docker/cli/v28/cli/command"
	"github.com/spf13/cobra"
)

// RemoveOptions contains options for the docker config rm command.
type RemoveOptions struct {
	Names []string
}

func newConfigRemoveCommand(dockerCli command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:     "rm CONFIG [CONFIG...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more configs",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := RemoveOptions{
				Names: args,
			}
			return RunConfigRemove(cmd.Context(), dockerCli, opts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeNames(dockerCli)(cmd, args, toComplete)
		},
	}
}

// RunConfigRemove removes the given Swarm configs.
func RunConfigRemove(ctx context.Context, dockerCLI command.Cli, opts RemoveOptions) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, name := range opts.Names {
		if err := apiClient.ConfigRemove(ctx, name); err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	}

	return errors.Join(errs...)
}
