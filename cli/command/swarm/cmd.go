package swarm

import (
	"github.com/spf13/cobra"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
)

// NewSwarmCommand returns a cobra command for `swarm` subcommands
func NewSwarmCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swarm",
		Short: "Manage Swarm",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "",
		},
	}
	cmd.AddCommand(
		newInitCommand(dockerCli),
		newJoinCommand(dockerCli),
		newJoinTokenCommand(dockerCli),
		newUnlockKeyCommand(dockerCli),
		newUpdateCommand(dockerCli),
		newLeaveCommand(dockerCli),
		newUnlockCommand(dockerCli),
		newCACommand(dockerCli),
	)
	return cmd
}
