package image

import (
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/command/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.RegisterCommand(newImageCommand)
}

// NewImageCommand returns a cobra command for `image` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewImageCommand(dockerCli command.Cli) *cobra.Command {
	return newImageCommand(dockerCli)
}

// newImageCommand returns a cobra command for `image` subcommands
func newImageCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Manage images",
		Args:  cli.NoArgs,
		RunE:  cli.ShowHelp(dockerCli.Err()),
	}
	cmd.AddCommand(
		newBuildCommand(dockerCli),
		newHistoryCommand(dockerCli),
		newImportCommand(dockerCli),
		newLoadCommand(dockerCli),
		newPullCommand(dockerCli),
		newPushCommand(dockerCli),
		newSaveCommand(dockerCli),
		newTagCommand(dockerCli),
		newListCommand(dockerCli),
		newImageRemoveCommand(dockerCli),
		newInspectCommand(dockerCli),
		newPruneCommand(dockerCli),
	)
	return cmd
}
