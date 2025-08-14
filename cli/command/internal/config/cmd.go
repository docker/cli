package config

import (
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/command/internal/commands"
	"github.com/moby/moby/api/types/swarm"
	"github.com/spf13/cobra"
)

func init() {
	commands.RegisterCommand(newConfigCommand)
}

// NewConfigCommand returns a cobra command for `config` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewConfigCommand(dockerCLI cli.Cli) *cobra.Command {
	return newConfigCommand(dockerCLI)
}

// newConfigCommand returns a cobra command for `config` subcommands
func newConfigCommand(dockerCLI cli.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Swarm configs",
		Args:  cli.NoArgs,
		RunE:  cli.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{
			"version": "1.30",
			"swarm":   "manager",
		},
	}
	cmd.AddCommand(
		newConfigListCommand(dockerCLI),
		newConfigCreateCommand(dockerCLI),
		newConfigInspectCommand(dockerCLI),
		newConfigRemoveCommand(dockerCLI),
	)
	return cmd
}

// completeNames offers completion for swarm configs
func completeNames(dockerCLI completion.APIClientProvider) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCLI.Client().ConfigList(cmd.Context(), swarm.ConfigListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, config := range list {
			names = append(names, config.ID)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
