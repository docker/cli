package network // import "docker.com/cli/v28/cli/command/network"

import (
	"github.com/docker/cli/v28/cli"
	"github.com/docker/cli/v28/cli/command"
	"github.com/spf13/cobra"
)

// NewNetworkCommand returns a cobra command for `network` subcommands
func NewNetworkCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "network",
		Short:       "Manage networks",
		Args:        cli.NoArgs,
		RunE:        command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{"version": "1.21"},
	}
	cmd.AddCommand(
		newConnectCommand(dockerCli),
		newCreateCommand(dockerCli),
		newDisconnectCommand(dockerCli),
		newInspectCommand(dockerCli),
		newListCommand(dockerCli),
		newRemoveCommand(dockerCli),
		NewPruneCommand(dockerCli),
	)
	return cmd
}
