package swarm

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type joinTokenOptions struct {
	role   string
	rotate bool
	quiet  bool
}

func newJoinTokenCommand(dockerCLI command.Cli) *cobra.Command {
	opts := joinTokenOptions{}

	cmd := &cobra.Command{
		Use:   "join-token [OPTIONS] (worker|manager)",
		Short: "Manage join tokens",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.role = args[0]
			return runJoinToken(cmd.Context(), dockerCLI, opts)
		},
		Annotations: map[string]string{
			"version": "1.24",
			"swarm":   "manager",
		},
		DisableFlagsInUseLine: true,
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
		res, err := apiClient.SwarmInspect(ctx, client.SwarmInspectOptions{})
		if err != nil {
			return err
		}

		_, err = apiClient.SwarmUpdate(ctx, client.SwarmUpdateOptions{
			Version:            res.Swarm.Version,
			Spec:               res.Swarm.Spec,
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
	res, err := apiClient.SwarmInspect(ctx, client.SwarmInspectOptions{})
	if err != nil {
		return err
	}

	if opts.quiet && worker {
		_, _ = fmt.Fprintln(dockerCLI.Out(), res.Swarm.JoinTokens.Worker)
		return nil
	}

	if opts.quiet && manager {
		_, _ = fmt.Fprintln(dockerCLI.Out(), res.Swarm.JoinTokens.Manager)
		return nil
	}

	infoResp, err := apiClient.Info(ctx, client.InfoOptions{})
	if err != nil {
		return err
	}

	return printJoinCommand(ctx, dockerCLI, infoResp.Info.Swarm.NodeID, worker, manager)
}

func printJoinCommand(ctx context.Context, dockerCLI command.Cli, nodeID string, worker bool, manager bool) error {
	apiClient := dockerCLI.Client()

	res, err := apiClient.NodeInspect(ctx, nodeID, client.NodeInspectOptions{})
	if err != nil {
		return err
	}

	sw, err := apiClient.SwarmInspect(ctx, client.SwarmInspectOptions{})
	if err != nil {
		return err
	}

	if res.Node.ManagerStatus != nil {
		if worker {
			_, _ = fmt.Fprintf(dockerCLI.Out(),
				"To add a worker to this swarm, run the following command:\n\n    docker swarm join --token %s %s\n\n",
				sw.Swarm.JoinTokens.Worker, res.Node.ManagerStatus.Addr,
			)
		}
		if manager {
			_, _ = fmt.Fprintf(dockerCLI.Out(),
				"To add a manager to this swarm, run the following command:\n\n    docker swarm join --token %s %s\n\n",
				sw.Swarm.JoinTokens.Manager, res.Node.ManagerStatus.Addr,
			)
		}
	}

	return nil
}
