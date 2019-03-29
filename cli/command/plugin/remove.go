package plugin

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

type rmOptions struct {
	force bool

	plugins []string
}

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	var opts rmOptions

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] PLUGIN [PLUGIN...]",
		Short:   "Remove one or more plugins",
		Aliases: []string{"remove"},
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.plugins = args
			return runRemove(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the removal of an active plugin")
	return cmd
}

func runRemove(dockerCli command.Cli, opts *rmOptions) error {
	ctx := context.Background()

	var errs cli.Errors
	for _, name := range opts.plugins {
		if err := dockerCli.Client().PluginRemove(ctx, name, types.PluginRemoveOptions{Force: opts.force}); err != nil {
			errs = append(errs, err)
			continue
		}
		fmt.Fprintln(dockerCli.Out(), name)
	}
	// Do not simplify to `return errs` because even if errs == nil, it is not a nil-error interface value.
	if errs != nil {
		return errs
	}
	return nil
}
