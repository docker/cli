package builder

import (
	"github.com/spf13/cobra"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
)

// NewBuilderCommand returns a cobra command for `builder` subcommands
func NewBuilderCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "builder",
		Short:       "Manage builds",
		Args:        cli.NoArgs,
		RunE:        command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{"version": "1.31"},
	}
	cmd.AddCommand(
		NewPruneCommand(dockerCli),
	)
	return cmd
}
