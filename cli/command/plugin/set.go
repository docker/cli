package plugin // import "docker.com/cli/v28/cli/command/plugin"

import (
	"github.com/docker/cli/v28/cli"
	"github.com/docker/cli/v28/cli/command"
	"github.com/spf13/cobra"
)

func newSetCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set PLUGIN KEY=VALUE [KEY=VALUE...]",
		Short: "Change settings for a plugin",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dockerCli.Client().PluginSet(cmd.Context(), args[0], args[1:])
		},
	}

	return cmd
}
