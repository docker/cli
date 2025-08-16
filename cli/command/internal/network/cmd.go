package network

import (
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/spf13/cobra"
)

// NewNetworkCommand returns a cobra command for `network` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewNetworkCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "network",
		Short:       "Manage networks",
		Args:        cli.NoArgs,
		RunE:        cli.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{"version": "1.21"},
	}
	cmd.AddCommand(
		newConnectCommand(dockerCli),
		newCreateCommand(dockerCli),
		newDisconnectCommand(dockerCli),
		newInspectCommand(dockerCli),
		newListCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newPruneCommand(dockerCli),
	)
	return cmd
}
