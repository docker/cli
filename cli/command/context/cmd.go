package context

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewContextCommand returns the context cli subcommand
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewContextCommand(dockerCLI command.Cli) *cobra.Command {
	return newContextCommand(dockerCLI)
}

// newContextCommand returns the context cli subcommand
func newContextCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage contexts",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCLI.Err()),
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
