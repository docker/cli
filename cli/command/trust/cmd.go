package trust

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.Register(newTrustCommand)
}

// NewTrustCommand returns a cobra command for `trust` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewTrustCommand(dockerCLI command.Cli) *cobra.Command {
	return newTrustCommand(dockerCLI)
}

func newTrustCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust",
		Short: "Manage trust on Docker images",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCLI.Err()),
	}
	cmd.AddCommand(
		newRevokeCommand(dockerCLI),
		newSignCommand(dockerCLI),
		newTrustKeyCommand(dockerCLI),
		newTrustSignerCommand(dockerCLI),
		newInspectCommand(dockerCLI),
	)
	return cmd
}
