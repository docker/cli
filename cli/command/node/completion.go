package node

import (
	"os"

	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// completeNodeNames offers completion for swarm node (host)names and optional IDs.
// By default, only names are returned.
// Set DOCKER_COMPLETION_SHOW_NODE_IDS=yes to also complete IDs.
//
// TODO(thaJeztah): add support for filters.
func completeNodeNames(dockerCLI completion.APIClientProvider) cobra.CompletionFunc {
	// https://github.com/docker/cli/blob/f9ced58158d5e0b358052432244b483774a1983d/contrib/completion/bash/docker#L41-L43
	showIDs := os.Getenv("DOCKER_COMPLETION_SHOW_NODE_IDS") == "yes"
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		res, err := dockerCLI.Client().NodeList(cmd.Context(), client.NodeListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		names := make([]string, 0, len(res.Items)+1)
		for _, node := range res.Items {
			if showIDs {
				names = append(names, node.Description.Hostname, node.ID)
			} else {
				names = append(names, node.Description.Hostname)
			}
		}
		// Nodes allow "self" as magic word for the current node.
		names = append(names, "self")
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
