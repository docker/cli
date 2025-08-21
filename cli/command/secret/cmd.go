package secret

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/internal/commands"
	"github.com/moby/moby/api/types/swarm"
	"github.com/spf13/cobra"
)

func init() {
	commands.Register(newSecretCommand)
}

// newSecretCommand returns a cobra command for `secret` subcommands.
func newSecretCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage Swarm secrets",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{
			"version": "1.25",
			"swarm":   "manager",
		},
	}
	cmd.AddCommand(
		newSecretListCommand(dockerCLI),
		newSecretCreateCommand(dockerCLI),
		newSecretInspectCommand(dockerCLI),
		newSecretRemoveCommand(dockerCLI),
	)
	return cmd
}

// completeNames offers completion for swarm secrets
func completeNames(dockerCLI completion.APIClientProvider) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCLI.Client().SecretList(cmd.Context(), swarm.SecretListOptions{})
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
