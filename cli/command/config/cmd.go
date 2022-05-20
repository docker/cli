package config

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

// NewConfigCommand returns a cobra command for `config` subcommands
func NewConfigCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Swarm configs",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			"version": "1.30",
			"swarm":   "manager",
		},
	}
	cmd.AddCommand(
		newConfigListCommand(dockerCli),
		newConfigCreateCommand(dockerCli),
		newConfigInspectCommand(dockerCli),
		newConfigRemoveCommand(dockerCli),
	)
	return cmd
}

// completeNames offers completion for swarm configs
func completeNames(dockerCli command.Cli) completion.ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCli.Client().ConfigList(cmd.Context(), types.ConfigListOptions{})
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
