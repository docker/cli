package node

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/moby/moby/api/types/swarm"
	"github.com/spf13/cobra"
)

func newPromoteCommand(dockerCLI cli.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "promote NODE [NODE...]",
		Short: "Promote one or more nodes to manager in the swarm",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPromote(cmd.Context(), dockerCLI, args)
		},
		ValidArgsFunction: completeNodeNames(dockerCLI),
	}
}

func runPromote(ctx context.Context, dockerCLI cli.Cli, nodes []string) error {
	promote := func(node *swarm.Node) error {
		if node.Spec.Role == swarm.NodeRoleManager {
			_, _ = fmt.Fprintf(dockerCLI.Out(), "Node %s is already a manager.\n", node.ID)
			return errNoRoleChange
		}
		node.Spec.Role = swarm.NodeRoleManager
		return nil
	}
	success := func(nodeID string) {
		_, _ = fmt.Fprintf(dockerCLI.Out(), "Node %s promoted to a manager in the swarm.\n", nodeID)
	}
	return updateNodes(ctx, dockerCLI, nodes, promote, success)
}
