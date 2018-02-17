package config

import (
	"github.com/spf13/cobra"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
)

// NewConfigCommand returns a cobra command for `config` subcommands
func NewConfigCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Docker configs",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			"version": "1.30",
			"swarm":   "",
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
