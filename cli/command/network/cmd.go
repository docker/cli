package network

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewNetworkCommand returns a cobra command for `network` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewNetworkCommand(dockerCLI command.Cli) *cobra.Command {
	return newNetworkCommand(dockerCLI)
}

// newNetworkCommand returns a cobra command for `network` subcommands
func newNetworkCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "network",
		Short:       "Manage networks",
		Args:        cli.NoArgs,
		RunE:        command.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{"version": "1.21"},
	}
	cmd.AddCommand(
		newConnectCommand(dockerCLI),
		newCreateCommand(dockerCLI),
		newDisconnectCommand(dockerCLI),
		newInspectCommand(dockerCLI),
		newListCommand(dockerCLI),
		newRemoveCommand(dockerCLI),
		newPruneCommand(dockerCLI),
	)
	return cmd
}
