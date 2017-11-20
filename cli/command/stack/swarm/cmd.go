package swarm

import (
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// AddStackCommands adds `stack` subcommands
func AddStackCommands(root *cobra.Command, dockerCli command.Cli) {
	root.AddCommand(
		newDeployCommand(dockerCli),
		newListCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newServicesCommand(dockerCli),
		newPsCommand(dockerCli),
	)
}

// NewTopLevelDeployCommand returns a command for `docker deploy`
func NewTopLevelDeployCommand(dockerCli command.Cli) *cobra.Command {
	return newDeployCommand(dockerCli)
}
