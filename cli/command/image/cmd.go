package image

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.Register(newBuildCommand)
	commands.Register(newPullCommand)
	commands.Register(newPushCommand)
	commands.Register(newImagesCommand)
	commands.Register(newImageCommand)
	commands.RegisterLegacy(newHistoryCommand)
	commands.RegisterLegacy(newImportCommand)
	commands.RegisterLegacy(newLoadCommand)
	commands.RegisterLegacy(newRemoveCommand)
	commands.RegisterLegacy(newSaveCommand)
	commands.RegisterLegacy(newTagCommand)
}

// newImageCommand returns a cobra command for `image` subcommands
func newImageCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Manage images",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
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
