package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	force bool
}

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	opts := removeOptions{}

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] NODE [NODE...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more nodes from the swarm",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), dockerCli, args, opts)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Force remove a node from the swarm")
	return cmd
}

func runRemove(ctx context.Context, dockerCLI command.Cli, nodeIDs []string, opts removeOptions) error {
	apiClient := dockerCLI.Client()

	var errs []error
	for _, id := range nodeIDs {
		if err := apiClient.NodeRemove(ctx, id, types.NodeRemoveOptions{Force: opts.force}); err != nil {
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), id)
	}
	return errors.Join(errs...)
}
