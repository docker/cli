package node

import (
	"context"
	"errors"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

// NewNodeCommand returns a cobra command for `node` subcommands
//
// Deprecated: Do not import commands directly. They will be removed in a future release.
func NewNodeCommand(dockerCLI command.Cli) *cobra.Command {
	return newNodeCommand(dockerCLI)
}

// newNodeCommand returns a cobra command for `node` subcommands
func newNodeCommand(dockerCLI command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage Swarm nodes",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCLI.Err()),
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "manager",
		},
	}
	cmd.AddCommand(
		newDemoteCommand(dockerCLI),
		newInspectCommand(dockerCLI),
		newListCommand(dockerCLI),
		newPromoteCommand(dockerCLI),
		newRemoveCommand(dockerCLI),
		newPsCommand(dockerCLI),
		newUpdateCommand(dockerCLI),
	)
	return cmd
}

// Reference returns the reference of a node. The special value "self" for a node
// reference is mapped to the current node, hence the node ID is retrieved using
// the `/info` endpoint.
func Reference(ctx context.Context, apiClient client.APIClient, ref string) (string, error) {
	if ref == "self" {
		info, err := apiClient.Info(ctx)
		if err != nil {
			return "", err
		}
		if info.Swarm.NodeID == "" {
			// If there's no node ID in /info, the node probably
			// isn't a manager. Call a swarm-specific endpoint to
			// get a more specific error message.
			//
			// FIXME(thaJeztah): this should not require calling a Swarm endpoint, and we could just suffice with info / ping (which has swarm status).
			_, err = apiClient.NodeList(ctx, swarm.NodeListOptions{})
			if err != nil {
				return "", err
			}
			return "", errors.New("node ID not found in /info")
		}
		return info.Swarm.NodeID, nil
	}
	return ref, nil
}
