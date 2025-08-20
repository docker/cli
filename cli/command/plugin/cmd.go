package plugin

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewPluginCommand returns a cobra command for `plugin` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewPluginCommand(dockerCLI command.Cli) *cobra.Command {
	return newPluginCommand(dockerCLI)
}

// newPluginCommand returns a cobra command for `plugin` subcommands
func newPluginCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "plugin",
		Short:       "Manage plugins",
		Args:        cli.NoArgs,
		RunE:        command.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{"version": "1.25"},
	}

	cmd.AddCommand(
		newDisableCommand(dockerCLI),
		newEnableCommand(dockerCLI),
		newInspectCommand(dockerCLI),
		newInstallCommand(dockerCLI),
		newListCommand(dockerCLI),
		newRemoveCommand(dockerCLI),
		newSetCommand(dockerCLI),
		newPushCommand(dockerCLI),
		newCreateCommand(dockerCLI),
		newUpgradeCommand(dockerCLI),
	)
	return cmd
}
