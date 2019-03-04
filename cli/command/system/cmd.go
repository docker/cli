package system

import (
	"runtime"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewSystemCommand returns a cobra command for `system` subcommands
func NewSystemCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "Manage Docker",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
	}
	cmd.AddCommand(
		NewEventsCommand(dockerCli),
		NewInfoCommand(dockerCli),
		newDiskUsageCommand(dockerCli),
		newPruneCommand(dockerCli),
	)
	if runtime.GOOS != "windows" {
		cmd.AddCommand(
			newDialStdioCommand(dockerCli),
		)
	}

	return cmd
}
