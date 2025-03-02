package swarm

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
)

// RunRemove is the swarm implementation of docker stack remove
func RunRemove(ctx context.Context, dockerCli command.Cli, opts options.Remove) error {
	apiClient := dockerCli.Client()

	var errs []error
	for _, namespace := range opts.Namespaces {
		services, err := getStackServices(ctx, apiClient, namespace)
		if err != nil {
			return err
		}

		networks, err := getStackNetworks(ctx, apiClient, namespace)
		if err != nil {
			return err
		}

		var secrets []swarm.Secret
		if versions.GreaterThanOrEqualTo(apiClient.ClientVersion(), "1.25") {
			secrets, err = getStackSecrets(ctx, apiClient, namespace)
			if err != nil {
				return err
			}
		}

		var configs []swarm.Config
		if versions.GreaterThanOrEqualTo(apiClient.ClientVersion(), "1.30") {
			configs, err = getStackConfigs(ctx, apiClient, namespace)
			if err != nil {
				return err
			}
		}

		if len(services)+len(networks)+len(secrets)+len(configs) == 0 {
			_, _ = fmt.Fprintln(dockerCli.Err(), "Nothing found in stack:", namespace)
			continue
		}

		// TODO(thaJeztah): change this "hasError" boolean to return a (multi-)error for each of these functions instead.
		hasError := removeServices(ctx, dockerCli, services)
		hasError = removeSecrets(ctx, dockerCli, secrets) || hasError
		hasError = removeConfigs(ctx, dockerCli, configs) || hasError
		hasError = removeNetworks(ctx, dockerCli, networks) || hasError

		if hasError {
			errs = append(errs, errors.New("failed to remove some resources from stack: "+namespace))
			continue
		}

		if !opts.Detach {
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
		if err := dockerCLI.Client().ServiceRemove(ctx, service.ID); err != nil {
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
		if err := dockerCLI.Client().NetworkRemove(ctx, nw.ID); err != nil {
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
		if err := dockerCli.Client().SecretRemove(ctx, secret.ID); err != nil {
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
		if err := dockerCLI.Client().ConfigRemove(ctx, config.ID); err != nil {
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
		tasks, err := getStackTasks(ctx, apiClient, namespace)
		if err != nil {
			return fmt.Errorf("failed to get tasks: %w", err)
		}

		for _, task := range tasks {
			if terminalState(task.Status.State) {
				terminalStatesReached++
				break
			}
		}

		if terminalStatesReached == len(tasks) {
			break
		}
	}
	return nil
}
