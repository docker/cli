package plugin

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func newSetCommand(dockerCLI command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "set PLUGIN KEY=VALUE [KEY=VALUE...]",
		Short: "Change settings for a plugin",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dockerCLI.Client().PluginSet(cmd.Context(), args[0], args[1:])
		},
		ValidArgsFunction:     completeNames(dockerCLI, stateAny), // TODO(thaJeztah): should only complete for the first arg
		DisableFlagsInUseLine: true,
	}
}
