package clustervolume

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"

	"github.com/spf13/cobra"
)

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:     "rm [VOLUME...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more cluster volumes",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dockerCli.Client().ClusterVolumeRemove(
				context.Background(), args[0],
			)
		},
	}
}
