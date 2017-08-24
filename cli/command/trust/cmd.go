package trust

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewTrustCommand returns a cobra command for `trust` subcommands
func NewTrustCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust",
		Short: "Sign images to establish trust",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
	}
	cmd.AddCommand(
		newInspectCommand(dockerCli),
		newRevokeCommand(dockerCli),
		newSignCommand(dockerCli),
	)
	return cmd
}
