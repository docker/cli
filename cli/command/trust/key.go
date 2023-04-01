package trust

import (
	"github.com/docker/cli/v24/cli"
	"github.com/docker/cli/v24/cli/command"
	"github.com/spf13/cobra"
)

// newTrustKeyCommand returns a cobra command for `trust key` subcommands
func newTrustKeyCommand(dockerCli command.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage keys for signing Docker images",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
	}
	cmd.AddCommand(
		newKeyGenerateCommand(dockerCli),
		newKeyLoadCommand(dockerCli),
	)
	return cmd
}
