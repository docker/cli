package checkpoint

import (
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/command/internal/commands"
	"github.com/spf13/cobra"
)

func init() {
	commands.RegisterCommand(newCheckpointCommand)
}

// NewCheckpointCommand returns the `checkpoint` subcommand (only in experimental)
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewCheckpointCommand(dockerCLI cli.Cli) *cobra.Command {
	return newCheckpointCommand(dockerCLI)
}

// newCheckpointCommand returns the `checkpoint` subcommand (only in experimental)
func newCheckpointCommand(dockerCLI cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkpoint",
		Short: "Manage checkpoints",
		Args:  cli.NoArgs,
		RunE:  cli.ShowHelp(dockerCLI.Err()),
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
