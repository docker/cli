package builder

import (
	"github.com/spf13/cobra"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/internal/commands"
)

func init() {
	commands.Register(newBuilderCommand)
	commands.Register(func(c command.Cli) *cobra.Command {
		return newBakeStubCommand(c)
	})
}

// newBuilderCommand returns a cobra command for `builder` subcommands
func newBuilderCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "builder",
		Short:       "Manage builds",
		Args:        cli.NoArgs,
		RunE:        command.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{"version": "1.31"},
	}
	cmd.AddCommand(
		NewPruneCommand(dockerCLI),
		// we should have a mechanism for registering sub-commands in the cli/internal/commands.Register function.
		//nolint:staticcheck // TODO: Remove when migration to cli/internal/commands.Register is complete. (see #6283)
		image.NewBuildCommand(dockerCLI),
	)
	return cmd
}

// newBakeStubCommand returns a cobra command "stub" for the "bake" subcommand.
// This command is a placeholder / stub that is dynamically replaced by an
// alias for "docker buildx bake" if BuildKit is enabled (and the buildx plugin
// installed).
func newBakeStubCommand(dockerCLI command.Streams) *cobra.Command {
	return &cobra.Command{
		Use:   "bake [OPTIONS] [TARGET...]",
		Short: "Build from a file",
		RunE:  command.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{
			// We want to show this command in the "top" category in --help
			// output, and not to be grouped under "management commands".
			"category-top": "5",
			"aliases":      "docker buildx bake",
			"version":      "1.31",
		},
	}
}
