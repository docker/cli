package container

import (
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/spf13/cobra"
)

// NewContainerCommand returns a cobra command for `container` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewContainerCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container",
		Short: "Manage containers",
		Args:  cli.NoArgs,
		RunE:  cli.ShowHelp(dockerCli.Err()),
	}
	cmd.AddCommand(
		newAttachCommand(dockerCli),
		newCommitCommand(dockerCli),
		newCopyCommand(dockerCli),
		newCreateCommand(dockerCli),
		newDiffCommand(dockerCli),
		newExecCommand(dockerCli),
		newExportCommand(dockerCli),
		newKillCommand(dockerCli),
		newLogsCommand(dockerCli),
		newPauseCommand(dockerCli),
		newPortCommand(dockerCli),
		newRenameCommand(dockerCli),
		newRestartCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newRunCommand(dockerCli),
		newStartCommand(dockerCli),
		newStatsCommand(dockerCli),
		newStopCommand(dockerCli),
		newTopCommand(dockerCli),
		newUnpauseCommand(dockerCli),
		newUpdateCommand(dockerCli),
		newWaitCommand(dockerCli),
		newListCommand(dockerCli),
		newInspectCommand(dockerCli),
		newPruneCommand(dockerCli),
	)
	return cmd
}
