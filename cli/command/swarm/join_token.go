package swarm

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type joinTokenOptions struct {
	role   string
	rotate bool
	quiet  bool
}

func newJoinTokenCommand(dockerCli command.Cli) *cobra.Command {
	opts := joinTokenOptions{}

	cmd := &cobra.Command{
		Use:   "join-token [OPTIONS] (worker|manager)",
		Short: "Manage join tokens",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.role = args[0]
			return runJoinToken(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "manager",
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.rotate, flagRotate, false, "Rotate join token")
	flags.BoolVarP(&opts.quiet, flagQuiet, "q", false, "Only display token")

	return cmd
}

func runJoinToken(ctx context.Context, dockerCLI command.Cli, opts joinTokenOptions) error {
	worker := opts.role == "worker"
	manager := opts.role == "manager"

	if !worker && !manager {
		return errors.New("unknown role " + opts.role)
	}

	apiClient := dockerCLI.Client()

	if opts.rotate {
		sw, err := apiClient.SwarmInspect(ctx)
		if err != nil {
			return err
		}

		err = apiClient.SwarmUpdate(ctx, sw.Version, sw.Spec, swarm.UpdateFlags{
			RotateWorkerToken:  worker,
			RotateManagerToken: manager,
		})
		if err != nil {
			return err
		}

		if !opts.quiet {
			_, _ = fmt.Fprintf(dockerCLI.Out(), "Successfully rotated %s join token.\n\n", opts.role)
		}
	}

	// second SwarmInspect in this function,
	// this is necessary since SwarmUpdate after first changes the join tokens
	sw, err := apiClient.SwarmInspect(ctx)
	if err != nil {
		return err
	}

	if opts.quiet && worker {
		_, _ = fmt.Fprintln(dockerCLI.Out(), sw.JoinTokens.Worker)
		return nil
	}

	if opts.quiet && manager {
		_, _ = fmt.Fprintln(dockerCLI.Out(), sw.JoinTokens.Manager)
		return nil
	}

	info, err := apiClient.Info(ctx)
	if err != nil {
		return err
	}

	return printJoinCommand(ctx, dockerCLI, info.Swarm.NodeID, worker, manager)
}

func printJoinCommand(ctx context.Context, dockerCLI command.Cli, nodeID string, worker bool, manager bool) error {
	apiClient := dockerCLI.Client()

	node, _, err := apiClient.NodeInspectWithRaw(ctx, nodeID)
	if err != nil {
		return err
	}

	sw, err := apiClient.SwarmInspect(ctx)
	if err != nil {
		return err
	}

	if node.ManagerStatus != nil {
		if worker {
			_, _ = fmt.Fprintf(dockerCLI.Out(), "To add a worker to this swarm, run the following command:\n\n    docker swarm join --token %s %s\n\n", sw.JoinTokens.Worker, node.ManagerStatus.Addr)
		}
		if manager {
			_, _ = fmt.Fprintf(dockerCLI.Out(), "To add a manager to this swarm, run the following command:\n\n    docker swarm join --token %s %s\n\n", sw.JoinTokens.Manager, node.ManagerStatus.Addr)
		}
	}

	return nil
}
