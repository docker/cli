package volume

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.Register(newVolumeCommand)
}

// NewVolumeCommand returns a cobra command for `volume` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewVolumeCommand(dockerCLI command.Cli) *cobra.Command {
	return newVolumeCommand(dockerCLI)
}

// newVolumeCommand returns a cobra command for `volume` subcommands
func newVolumeCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "volume COMMAND",
		Short:       "Manage volumes",
		Args:        cli.NoArgs,
		RunE:        command.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{"version": "1.21"},
	}
	cmd.AddCommand(
		newCreateCommand(dockerCLI),
		newInspectCommand(dockerCLI),
		newListCommand(dockerCLI),
		newRemoveCommand(dockerCLI),
		newPruneCommand(dockerCLI),
		newUpdateCommand(dockerCLI),
	)
	return cmd
}
