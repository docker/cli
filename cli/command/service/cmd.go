package service

import (
	"os"

	"github.com/docker/cli/v28/cli"
	"github.com/docker/cli/v28/cli/command"
	"github.com/docker/cli/v28/cli/command/completion"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

// NewServiceCommand returns a cobra command for `service` subcommands
func NewServiceCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage Swarm services",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "manager",
		},
	}
	cmd.AddCommand(
		newCreateCommand(dockerCli),
		newInspectCommand(dockerCli),
		newPsCommand(dockerCli),
		newListCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newScaleCommand(dockerCli),
		newUpdateCommand(dockerCli),
		newLogsCommand(dockerCli),
		newRollbackCommand(dockerCli),
	)
	return cmd
}

// CompletionFn offers completion for swarm service names and optional IDs.
// By default, only names are returned.
// Set DOCKER_COMPLETION_SHOW_SERVICE_IDS=yes to also complete IDs.
func CompletionFn(dockerCLI completion.APIClientProvider) completion.ValidArgsFn {
	// https://github.com/docker/cli/blob/f9ced58158d5e0b358052432244b483774a1983d/contrib/completion/bash/docker#L41-L43
	showIDs := os.Getenv("DOCKER_COMPLETION_SHOW_SERVICE_IDS") == "yes"
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCLI.Client().ServiceList(cmd.Context(), types.ServiceListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		names := make([]string, 0, len(list))
		for _, service := range list {
			if showIDs {
				names = append(names, service.Spec.Name, service.ID)
			} else {
				names = append(names, service.Spec.Name)
			}
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
