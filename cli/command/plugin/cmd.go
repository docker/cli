package plugin

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewPluginCommand returns a cobra command for `plugin` subcommands
func NewPluginCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "plugin",
		Short:       "Manage plugins",
		Args:        cli.NoArgs,
		RunE:        command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{"version": "1.25"},
	}

	cmd.AddCommand(
		newDisableCommand(dockerCli),
		newEnableCommand(dockerCli),
		newInspectCommand(dockerCli),
		newInstallCommand(dockerCli),
		newListCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newSetCommand(dockerCli),
		newPushCommand(dockerCli),
		newCreateCommand(dockerCli),
		newUpgradeCommand(dockerCli),
	)
	return cmd
}
