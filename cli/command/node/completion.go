package node // import "docker.com/cli/v28/cli/command/node"

import (
	"os"

	"github.com/docker/cli/v28/cli/command/completion"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

// completeNodeNames offers completion for swarm node (host)names and optional IDs.
// By default, only names are returned.
// Set DOCKER_COMPLETION_SHOW_NODE_IDS=yes to also complete IDs.
//
// TODO(thaJeztah): add support for filters.
func completeNodeNames(dockerCLI completion.APIClientProvider) completion.ValidArgsFn {
	// https://github.com/docker/cli/blob/f9ced58158d5e0b358052432244b483774a1983d/contrib/completion/bash/docker#L41-L43
	showIDs := os.Getenv("DOCKER_COMPLETION_SHOW_NODE_IDS") == "yes"
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		list, err := dockerCLI.Client().NodeList(cmd.Context(), types.NodeListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		names := make([]string, 0, len(list)+1)
		for _, node := range list {
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
