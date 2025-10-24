package plugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type rmOptions struct {
	force bool

	plugins []string
}

func newRemoveCommand(dockerCLI command.Cli) *cobra.Command {
	var opts rmOptions

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] PLUGIN [PLUGIN...]",
		Short:   "Remove one or more plugins",
		Aliases: []string{"remove"},
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.plugins = args
			return runRemove(cmd.Context(), dockerCLI, &opts)
		},
		ValidArgsFunction:     completeNames(dockerCLI, stateAny),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the removal of an active plugin")
	return cmd
}

func runRemove(ctx context.Context, dockerCLI command.Cli, opts *rmOptions) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, name := range opts.plugins {
		if _, err := apiClient.PluginRemove(ctx, name, client.PluginRemoveOptions{Force: opts.force}); err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	}
	return errors.Join(errs...)
}
