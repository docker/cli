package stack

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

// removeOptions holds docker stack remove options
type removeOptions struct {
	namespaces []string
	detach     bool
}

func newRemoveCommand(dockerCLI command.Cli) *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] STACK [STACK...]",
		Aliases: []string{"remove", "down"},
		Short:   "Remove one or more stacks",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.namespaces = args
			if err := validateStackNames(opts.namespaces); err != nil {
				return err
			}
			return runRemove(cmd.Context(), dockerCLI, opts)
		},
		ValidArgsFunction:     completeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.detach, "detach", "d", true, "Do not wait for stack removal")
	return cmd
}

// runRemove is the swarm implementation of docker stack remove.
func runRemove(ctx context.Context, dockerCli command.Cli, opts removeOptions) error {
	apiClient := dockerCli.Client()

	var errs []error
	for _, namespace := range opts.namespaces {
		services, err := getStackServices(ctx, apiClient, namespace)
		if err != nil {
			return err
		}

		networks, err := getStackNetworks(ctx, apiClient, namespace)
		if err != nil {
			return err
		}

		secrets, err := getStackSecrets(ctx, apiClient, namespace)
		if err != nil {
			return err
		}

		configs, err := getStackConfigs(ctx, apiClient, namespace)
		if err != nil {
			return err
		}

		if len(services.Items)+len(networks.Items)+len(secrets.Items)+len(configs.Items) == 0 {
			_, _ = fmt.Fprintln(dockerCli.Err(), "Nothing found in stack:", namespace)
			continue
		}

		// TODO(thaJeztah): change this "hasError" boolean to return a (multi-)error for each of these functions instead.
		hasError := removeServices(ctx, dockerCli, services.Items)
		hasError = removeSecrets(ctx, dockerCli, secrets.Items) || hasError
		hasError = removeConfigs(ctx, dockerCli, configs.Items) || hasError
		hasError = removeNetworks(ctx, dockerCli, networks.Items) || hasError

		if hasError {
			errs = append(errs, errors.New("failed to remove some resources from stack: "+namespace))
			continue
		}

		if !opts.detach {
			err = waitOnTasks(ctx, apiClient, namespace)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to wait on tasks of stack: %s: %w", namespace, err))
			}
		}
	}
	return errors.Join(errs...)
}

func sortServiceByName(services []swarm.Service) func(i, j int) bool {
	return func(i, j int) bool {
		return services[i].Spec.Name < services[j].Spec.Name
	}
}

func removeServices(ctx context.Context, dockerCLI command.Cli, services []swarm.Service) bool {
	var hasError bool
	sort.Slice(services, sortServiceByName(services))
	for _, service := range services {
		_, _ = fmt.Fprintln(dockerCLI.Out(), "Removing service", service.Spec.Name)
		if _, err := dockerCLI.Client().ServiceRemove(ctx, service.ID, client.ServiceRemoveOptions{}); err != nil {
			hasError = true
			_, _ = fmt.Fprintf(dockerCLI.Err(), "Failed to remove service %s: %s", service.ID, err)
		}
	}
	return hasError
}

func removeNetworks(ctx context.Context, dockerCLI command.Cli, networks []network.Summary) bool {
	var hasError bool
	for _, nw := range networks {
		_, _ = fmt.Fprintln(dockerCLI.Out(), "Removing network", nw.Name)
		if _, err := dockerCLI.Client().NetworkRemove(ctx, nw.ID, client.NetworkRemoveOptions{}); err != nil {
			hasError = true
			_, _ = fmt.Fprintf(dockerCLI.Err(), "Failed to remove network %s: %s", nw.ID, err)
		}
	}
	return hasError
}

func removeSecrets(ctx context.Context, dockerCli command.Cli, secrets []swarm.Secret) bool {
	var hasError bool
	for _, secret := range secrets {
		_, _ = fmt.Fprintln(dockerCli.Out(), "Removing secret", secret.Spec.Name)
		if _, err := dockerCli.Client().SecretRemove(ctx, secret.ID, client.SecretRemoveOptions{}); err != nil {
			hasError = true
			_, _ = fmt.Fprintf(dockerCli.Err(), "Failed to remove secret %s: %s", secret.ID, err)
		}
	}
	return hasError
}

func removeConfigs(ctx context.Context, dockerCLI command.Cli, configs []swarm.Config) bool {
	var hasError bool
	for _, config := range configs {
		_, _ = fmt.Fprintln(dockerCLI.Out(), "Removing config", config.Spec.Name)
		if _, err := dockerCLI.Client().ConfigRemove(ctx, config.ID, client.ConfigRemoveOptions{}); err != nil {
			hasError = true
			_, _ = fmt.Fprintf(dockerCLI.Err(), "Failed to remove config %s: %s", config.ID, err)
		}
	}
	return hasError
}

var numberedStates = map[swarm.TaskState]int64{
	swarm.TaskStateNew:       1,
	swarm.TaskStateAllocated: 2,
	swarm.TaskStatePending:   3,
	swarm.TaskStateAssigned:  4,
	swarm.TaskStateAccepted:  5,
	swarm.TaskStatePreparing: 6,
	swarm.TaskStateReady:     7,
	swarm.TaskStateStarting:  8,
	swarm.TaskStateRunning:   9,
	swarm.TaskStateComplete:  10,
	swarm.TaskStateShutdown:  11,
	swarm.TaskStateFailed:    12,
	swarm.TaskStateRejected:  13,
}

func terminalState(state swarm.TaskState) bool {
	return numberedStates[state] > numberedStates[swarm.TaskStateRunning]
}

func waitOnTasks(ctx context.Context, apiClient client.APIClient, namespace string) error {
	terminalStatesReached := 0
	for {
		res, err := getStackTasks(ctx, apiClient, namespace)
		if err != nil {
			return fmt.Errorf("failed to get tasks: %w", err)
		}

		for _, task := range res.Items {
			if terminalState(task.Status.State) {
				terminalStatesReached++
				break
			}
		}

		if terminalStatesReached == len(res.Items) {
			break
		}
	}
	return nil
}
