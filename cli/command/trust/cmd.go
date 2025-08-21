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

// newTrustCommand returns a cobra command for `trust` subcommands.
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
