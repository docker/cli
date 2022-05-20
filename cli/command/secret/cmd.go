package secret

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

// NewSecretCommand returns a cobra command for `secret` subcommands
func NewSecretCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage Swarm secrets",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			"version": "1.25",
			"swarm":   "manager",
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

// completeNames offers completion for swarm secrets
func completeNames(dockerCli command.Cli) completion.ValidArgsFn {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCli.Client().SecretList(cmd.Context(), types.SecretListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, secret := range list {
			names = append(names, secret.ID)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
