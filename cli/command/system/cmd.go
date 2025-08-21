package system

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.Register(newVersionCommand)
	commands.Register(newInfoCommand)
	commands.Register(newSystemCommand)
	commands.RegisterLegacy(newEventsCommand)
	commands.RegisterLegacy(newInspectCommand)
}

// newSystemCommand returns a cobra command for `system` subcommands
func newSystemCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "Manage Docker",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCLI.Err()),
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
