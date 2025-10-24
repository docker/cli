package service

import (
	"os"

	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// completeServiceNames offers completion for swarm service names and optional IDs.
// By default, only names are returned.
// Set DOCKER_COMPLETION_SHOW_SERVICE_IDS=yes to also complete IDs.
func completeServiceNames(dockerCLI completion.APIClientProvider) cobra.CompletionFunc {
	// https://github.com/docker/cli/blob/f9ced58158d5e0b358052432244b483774a1983d/contrib/completion/bash/docker#L41-L43
	showIDs := os.Getenv("DOCKER_COMPLETION_SHOW_SERVICE_IDS") == "yes"
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		res, err := dockerCLI.Client().ServiceList(cmd.Context(), client.ServiceListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		names := make([]string, 0, len(res.Items))
		for _, service := range res.Items {
			if showIDs {
				names = append(names, service.Spec.Name, service.ID)
			} else {
				names = append(names, service.Spec.Name)
			}
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
