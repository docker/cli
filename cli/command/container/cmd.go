package container

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.Register(newRunCommand)
	commands.Register(newExecCommand)
	commands.Register(newPsCommand)
	commands.Register(newContainerCommand)
	commands.RegisterLegacy(newAttachCommand)
	commands.RegisterLegacy(newCommitCommand)
	commands.RegisterLegacy(newCopyCommand)
	commands.RegisterLegacy(newCreateCommand)
	commands.RegisterLegacy(newDiffCommand)
	commands.RegisterLegacy(newExportCommand)
	commands.RegisterLegacy(newKillCommand)
	commands.RegisterLegacy(newLogsCommand)
	commands.RegisterLegacy(newPauseCommand)
	commands.RegisterLegacy(newPortCommand)
	commands.RegisterLegacy(newRenameCommand)
	commands.RegisterLegacy(newRestartCommand)
	commands.RegisterLegacy(newRmCommand)
	commands.RegisterLegacy(newStartCommand)
	commands.RegisterLegacy(newStatsCommand)
	commands.RegisterLegacy(newStopCommand)
	commands.RegisterLegacy(newTopCommand)
	commands.RegisterLegacy(newUnpauseCommand)
	commands.RegisterLegacy(newUpdateCommand)
	commands.RegisterLegacy(newWaitCommand)
}

// newContainerCommand returns a cobra command for `container` subcommands
func newContainerCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container",
		Short: "Manage containers",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCLI.Err()),
	}
	cmd.AddCommand(
		newAttachCommand(dockerCLI),
		newCommitCommand(dockerCLI),
		newCopyCommand(dockerCLI),
		newCreateCommand(dockerCLI),
		newDiffCommand(dockerCLI),
		newExecCommand(dockerCLI),
		newExportCommand(dockerCLI),
		newKillCommand(dockerCLI),
		newLogsCommand(dockerCLI),
		newPauseCommand(dockerCLI),
		newPortCommand(dockerCLI),
		newRenameCommand(dockerCLI),
		newRestartCommand(dockerCLI),
		newRemoveCommand(dockerCLI),
		newRunCommand(dockerCLI),
		newStartCommand(dockerCLI),
		newStatsCommand(dockerCLI),
		newStopCommand(dockerCLI),
		newTopCommand(dockerCLI),
		newUnpauseCommand(dockerCLI),
		newUpdateCommand(dockerCLI),
		newWaitCommand(dockerCLI),
		newListCommand(dockerCLI),
		newInspectCommand(dockerCLI),
		newPruneCommand(dockerCLI),
	)
	return cmd
}
