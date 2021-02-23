package clustervolume

import (
	"github.com/spf13/cobra"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
)

func NewClusterVolumeCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster-volume",
		Short: "Manage Swarm cluster volumes",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			// TODO(dperny): temporarily set to 1.41 for testing purposes.
			"version": "1.41",
			"swarm":   "",
		},
	}

	cmd.AddCommand(
		newCreateCommand(dockerCli),
		newUpdateCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newInspectCommand(dockerCli),
		newListCommand(dockerCli),

		// TODO(dperny): These are temporary testing commands
		newAddDriverCommand(dockerCli),
		newRemoveDriverCommand(dockerCli),
	)

	return cmd
}
