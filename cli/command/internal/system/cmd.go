package system

import (
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/command/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.RegisterCommand(newSystemCommand)
}

// NewSystemCommand returns a cobra command for `system` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewSystemCommand(dockerCLI cli.Cli) *cobra.Command {
	return newSystemCommand(dockerCLI)
}

// NewSystemCommand returns a cobra command for `system` subcommands
func newSystemCommand(dockerCLI cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "Manage Docker",
		Args:  cli.NoArgs,
		RunE:  cli.ShowHelp(dockerCLI.Err()),
	}
	cmd.AddCommand(
		newEventsCommand(dockerCLI),
		newInfoCommand(dockerCLI),
		newDiskUsageCommand(dockerCLI),
		newPruneCommand(dockerCLI),
		newDialStdioCommand(dockerCLI),
	)

	return cmd
}
