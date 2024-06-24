package swarm

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli/command"
	servicecli "github.com/docker/cli/cli/command/service"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/compose/convert"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
)

func deployCompose(ctx context.Context, dockerCli command.Cli, opts *options.Deploy, config *composetypes.Config) error {
	if err := checkDaemonIsSwarmManager(ctx, dockerCli); err != nil {
		return err
	}

	namespace := convert.NewNamespace(opts.Namespace)

	if opts.Prune {
		services := map[string]struct{}{}
		for _, service := range config.Services {
			services[service.Name] = struct{}{}
		}
		pruneServices(ctx, dockerCli, namespace, services)
	}

	serviceNetworks := getServicesDeclaredNetworks(config.Services)
	networks, externalNetworks := convert.Networks(namespace, config.Networks, serviceNetworks)
	if err := validateExternalNetworks(ctx, dockerCli.Client(), externalNetworks); err != nil {
		return err
	}
	if err := createNetworks(ctx, dockerCli, namespace, networks); err != nil {
		return err
	}

	secrets, err := convert.Secrets(namespace, config.Secrets)
	if err != nil {
		return err
	}
	if err := createSecrets(ctx, dockerCli, secrets); err != nil {
		return err
	}

	configs, err := convert.Configs(namespace, config.Configs)
	if err != nil {
		return err
	}
	if err := createConfigs(ctx, dockerCli, configs); err != nil {
		return err
	}

	services, err := convert.Services(ctx, namespace, config, dockerCli.Client())
	if err != nil {
		return err
	}

	serviceIDs, err := deployServices(ctx, dockerCli, services, namespace, opts.SendRegistryAuth, opts.ResolveImage)
	if err != nil {
		return err
	}

	if opts.Detach {
		return nil
	}

	return waitOnServices(ctx, dockerCli, serviceIDs, opts.Quiet)
}

func getServicesDeclaredNetworks(serviceConfigs []composetypes.ServiceConfig) map[string]struct{} {
	serviceNetworks := map[string]struct{}{}
	for _, serviceConfig := range serviceConfigs {
		if len(serviceConfig.Networks) == 0 {
			serviceNetworks["default"] = struct{}{}
			continue
		}
		for nw := range serviceConfig.Networks {
			serviceNetworks[nw] = struct{}{}
		}
	}
	return serviceNetworks
}

func validateExternalNetworks(ctx context.Context, apiClient client.NetworkAPIClient, externalNetworks []string) error {
	for _, networkName := range externalNetworks {
		if !container.NetworkMode(networkName).IsUserDefined() {
			// Networks that are not user defined always exist on all nodes as
			// local-scoped networks, so there's no need to inspect them.
			continue
		}
		nw, err := apiClient.NetworkInspect(ctx, networkName, network.InspectOptions{})
		switch {
		case errdefs.IsNotFound(err):
			return fmt.Errorf("network %q is declared as external, but could not be found. You need to create a swarm-scoped network before the stack is deployed", networkName)
		case err != nil:
			return err
		case nw.Scope != "swarm":
			return fmt.Errorf("network %q is declared as external, but it is not in the right scope: %q instead of \"swarm\"", networkName, nw.Scope)
		}
	}
	return nil
}

func createSecrets(ctx context.Context, dockerCli command.Cli, secrets []swarm.SecretSpec) error {
	apiClient := dockerCli.Client()

	for _, secretSpec := range secrets {
		secret, _, err := apiClient.SecretInspectWithRaw(ctx, secretSpec.Name)
		switch {
		case err == nil:
			// secret already exists, then we update that
			if err := apiClient.SecretUpdate(ctx, secret.ID, secret.Meta.Version, secretSpec); err != nil {
				return fmt.Errorf("failed to update secret %s: %w", secretSpec.Name, err)
			}
		case errdefs.IsNotFound(err):
			// secret does not exist, then we create a new one.
			fmt.Fprintf(dockerCli.Out(), "Creating secret %s\n", secretSpec.Name)
			if _, err := apiClient.SecretCreate(ctx, secretSpec); err != nil {
				return fmt.Errorf("failed to create secret %s: %w", secretSpec.Name, err)
			}
		default:
			return err
		}
	}
	return nil
}

func createConfigs(ctx context.Context, dockerCli command.Cli, configs []swarm.ConfigSpec) error {
	apiClient := dockerCli.Client()

	for _, configSpec := range configs {
		config, _, err := apiClient.ConfigInspectWithRaw(ctx, configSpec.Name)
		switch {
		case err == nil:
			// config already exists, then we update that
			if err := apiClient.ConfigUpdate(ctx, config.ID, config.Meta.Version, configSpec); err != nil {
				return fmt.Errorf("failed to update config %s: %w", configSpec.Name, err)
			}
		case errdefs.IsNotFound(err):
			// config does not exist, then we create a new one.
			fmt.Fprintf(dockerCli.Out(), "Creating config %s\n", configSpec.Name)
			if _, err := apiClient.ConfigCreate(ctx, configSpec); err != nil {
				return fmt.Errorf("failed to create config %s: %w", configSpec.Name, err)
			}
		default:
			return err
		}
	}
	return nil
}

func createNetworks(ctx context.Context, dockerCli command.Cli, namespace convert.Namespace, networks map[string]network.CreateOptions) error {
	apiClient := dockerCli.Client()

	existingNetworks, err := getStackNetworks(ctx, apiClient, namespace.Name())
	if err != nil {
		return err
	}

	existingNetworkMap := make(map[string]network.Summary)
	for _, nw := range existingNetworks {
		existingNetworkMap[nw.Name] = nw
	}

	for name, createOpts := range networks {
		if _, exists := existingNetworkMap[name]; exists {
			continue
		}

		if createOpts.Driver == "" {
			createOpts.Driver = defaultNetworkDriver
		}

		fmt.Fprintf(dockerCli.Out(), "Creating network %s\n", name)
		if _, err := apiClient.NetworkCreate(ctx, name, createOpts); err != nil {
			return fmt.Errorf("failed to create network %s: %w", name, err)
		}
	}
	return nil
}

func deployServices(ctx context.Context, dockerCli command.Cli, services map[string]swarm.ServiceSpec, namespace convert.Namespace, sendAuth bool, resolveImage string) ([]string, error) {
	apiClient := dockerCli.Client()
	out := dockerCli.Out()

	existingServices, err := getStackServices(ctx, apiClient, namespace.Name())
	if err != nil {
		return nil, err
	}

	existingServiceMap := make(map[string]swarm.Service)
	for _, service := range existingServices {
		existingServiceMap[service.Spec.Name] = service
	}

	var serviceIDs []string

	for internalName, serviceSpec := range services {
		var (
			name        = namespace.Scope(internalName)
			image       = serviceSpec.TaskTemplate.ContainerSpec.Image
			encodedAuth string
		)

		if sendAuth {
			// Retrieve encoded auth token from the image reference
			encodedAuth, err = command.RetrieveAuthTokenFromImage(dockerCli.ConfigFile(), image)
			if err != nil {
				return nil, err
			}
		}

		if service, exists := existingServiceMap[name]; exists {
			fmt.Fprintf(out, "Updating service %s (id: %s)\n", name, service.ID)

			updateOpts := types.ServiceUpdateOptions{EncodedRegistryAuth: encodedAuth}

			switch resolveImage {
			case ResolveImageAlways:
				// image should be updated by the server using QueryRegistry
				updateOpts.QueryRegistry = true
			case ResolveImageChanged:
				if image != service.Spec.Labels[convert.LabelImage] {
					// Query the registry to resolve digest for the updated image
					updateOpts.QueryRegistry = true
				} else {
					// image has not changed; update the serviceSpec with the
					// existing information that was set by QueryRegistry on the
					// previous deploy. Otherwise this will trigger an incorrect
					// service update.
					serviceSpec.TaskTemplate.ContainerSpec.Image = service.Spec.TaskTemplate.ContainerSpec.Image
				}
			default:
				if image == service.Spec.Labels[convert.LabelImage] {
					// image has not changed; update the serviceSpec with the
					// existing information that was set by QueryRegistry on the
					// previous deploy. Otherwise this will trigger an incorrect
					// service update.
					serviceSpec.TaskTemplate.ContainerSpec.Image = service.Spec.TaskTemplate.ContainerSpec.Image
				}
			}

			// Stack deploy does not have a `--force` option. Preserve existing
			// ForceUpdate value so that tasks are not re-deployed if not updated.
			// TODO move this to API client?
			serviceSpec.TaskTemplate.ForceUpdate = service.Spec.TaskTemplate.ForceUpdate

			response, err := apiClient.ServiceUpdate(ctx, service.ID, service.Version, serviceSpec, updateOpts)
			if err != nil {
				return nil, fmt.Errorf("failed to update service %s: %w", name, err)
			}

			for _, warning := range response.Warnings {
				fmt.Fprintln(dockerCli.Err(), warning)
			}

			serviceIDs = append(serviceIDs, service.ID)
		} else {
			fmt.Fprintf(out, "Creating service %s\n", name)

			createOpts := types.ServiceCreateOptions{EncodedRegistryAuth: encodedAuth}

			// query registry if flag disabling it was not set
			if resolveImage == ResolveImageAlways || resolveImage == ResolveImageChanged {
				createOpts.QueryRegistry = true
			}

			response, err := apiClient.ServiceCreate(ctx, serviceSpec, createOpts)
			if err != nil {
				return nil, fmt.Errorf("failed to create service %s: %w", name, err)
			}

			serviceIDs = append(serviceIDs, response.ID)
		}
	}

	return serviceIDs, nil
}

func waitOnServices(ctx context.Context, dockerCli command.Cli, serviceIDs []string, quiet bool) error {
	var errs []error
	for _, serviceID := range serviceIDs {
		if err := servicecli.WaitOnService(ctx, dockerCli, serviceID, quiet); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", serviceID, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
