package secret

import (
	"github.com/spf13/cobra"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
)

// NewSecretCommand returns a cobra command for `secret` subcommands
func NewSecretCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage Docker secrets",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			"version": "1.25",
			"swarm":   "",
		},
	}
	cmd.AddCommand(
		newSecretListCommand(dockerCli),
		newSecretCreateCommand(dockerCli),
		newSecretInspectCommand(dockerCli),
		newSecretRemoveCommand(dockerCli),
	)
	return cmd
}
