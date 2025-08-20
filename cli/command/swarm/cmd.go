package swarm

import (
	"github.com/spf13/cobra"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
)

// NewSwarmCommand returns a cobra command for `swarm` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewSwarmCommand(dockerCLI command.Cli) *cobra.Command {
	return newSwarmCommand(dockerCLI)
}

// newSwarmCommand returns a cobra command for `swarm` subcommands
func newSwarmCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swarm",
		Short: "Manage Swarm",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "", // swarm command itself does not require swarm to be enabled (so swarm init and join is always available on API 1.24 and up)
		},
	}
	cmd.AddCommand(
		newInitCommand(dockerCLI),
		newJoinCommand(dockerCLI),
		newJoinTokenCommand(dockerCLI),
		newUnlockKeyCommand(dockerCLI),
		newUpdateCommand(dockerCLI),
		newLeaveCommand(dockerCLI),
		newUnlockCommand(dockerCLI),
		newCACommand(dockerCLI),
	)
	return cmd
}
