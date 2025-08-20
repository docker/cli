package checkpoint

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.Register(newCheckpointCommand)
}

// NewCheckpointCommand returns the `checkpoint` subcommand (only in experimental)
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewCheckpointCommand(dockerCLI command.Cli) *cobra.Command {
	return newCheckpointCommand(dockerCLI)
}

func newCheckpointCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkpoint",
		Short: "Manage checkpoints",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{
			"experimental": "",
			"ostype":       "linux",
			"version":      "1.25",
		},
	}
	cmd.AddCommand(
		newCreateCommand(dockerCLI),
		newListCommand(dockerCLI),
		newRemoveCommand(dockerCLI),
	)
	return cmd
}
