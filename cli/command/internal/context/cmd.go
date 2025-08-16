package context

import (
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/command/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.RegisterCommand(newContextCommand)
}

// NewContextCommand returns the context cli subcommand
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewContextCommand(dockerCLI cli.Cli) *cobra.Command {
	return newContextCommand(dockerCLI)
}

// newContextCommand returns the context cli subcommand
func newContextCommand(dockerCLI cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage contexts",
		Args:  cli.NoArgs,
		RunE:  cli.ShowHelp(dockerCLI.Err()),
	}
	cmd.AddCommand(
		newCreateCommand(dockerCLI),
		newListCommand(dockerCLI),
		newUseCommand(dockerCLI),
		newExportCommand(dockerCLI),
		newImportCommand(dockerCLI),
		newRemoveCommand(dockerCLI),
		newUpdateCommand(dockerCLI),
		newInspectCommand(dockerCLI),
		newShowCommand(dockerCLI),
	)
	return cmd
}
