package trust

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// newTrustSignerCommand returns a cobra command for `trust signer` subcommands
func newTrustSignerCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signer",
		Short: "Manage entities who can sign Docker images",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
	}
	cmd.AddCommand(
		newSignerAddCommand(dockerCli),
		newSignerRemoveCommand(dockerCli),
	)
	return cmd
}
