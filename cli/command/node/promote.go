package node

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/api/types/swarm"
	"github.com/spf13/cobra"
)

func newPromoteCommand(dockerCLI command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "promote NODE [NODE...]",
		Short: "Promote one or more nodes to manager in the swarm",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPromote(cmd.Context(), dockerCLI, args)
		},
		ValidArgsFunction:     completeNodeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}
}

func runPromote(ctx context.Context, dockerCLI command.Cli, nodes []string) error {
	promote := func(node *swarm.Node) error {
		if node.Spec.Role == swarm.NodeRoleManager {
			_, _ = fmt.Fprintf(dockerCLI.Out(), "Node %s is already a manager.\n", node.ID)
			return errNoRoleChange
		}
		node.Spec.Role = swarm.NodeRoleManager
		return nil
	}
	return updateNodes(ctx, dockerCLI.Client(), nodes, promote, func(nodeID string) {
		_, _ = fmt.Fprintf(dockerCLI.Out(), "Node %s promoted to a manager in the swarm.\n", nodeID)
	})
}
