package stack

import (
	"context"
	"errors"
	"fmt"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/service"
	"github.com/docker/cli/cli/compose/convert"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

func deployCompose(ctx context.Context, dockerCli command.Cli, opts *deployOptions, config *composetypes.Config) error {
	if err := checkDaemonIsSwarmManager(ctx, dockerCli); err != nil {
		return err
	}

	namespace := convert.NewNamespace(opts.namespace)

	if opts.prune {
		services := map[string]struct{}{}
		for _, svc := range config.Services {
			services[svc.Name] = struct{}{}
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

	serviceIDs, err := deployServices(ctx, dockerCli, services, namespace, opts.sendRegistryAuth, opts.resolveImage)
	if err != nil {
		return err
	}

	if opts.detach {
		return nil
	}

	return waitOnServices(ctx, dockerCli, serviceIDs, opts.quiet)
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
		res, err := apiClient.NetworkInspect(ctx, networkName, client.NetworkInspectOptions{})
		switch {
		case errdefs.IsNotFound(err):
			return fmt.Errorf("network %q is declared as external, but could not be found. You need to create a swarm-scoped network before the stack is deployed", networkName)
		case err != nil:
			return err
		case res.Network.Scope != "swarm":
			return fmt.Errorf("network %q is declared as external, but it is not in the right scope: %q instead of \"swarm\"", networkName, res.Network.Scope)
		}
	}
	return nil
}

func createSecrets(ctx context.Context, dockerCLI command.Cli, secrets []swarm.SecretSpec) error {
	apiClient := dockerCLI.Client()

	for _, secretSpec := range secrets {
		res, err := apiClient.SecretInspect(ctx, secretSpec.Name, client.SecretInspectOptions{})
		switch {
		case err == nil:
			// secret already exists, then we update that
			_, err := apiClient.SecretUpdate(ctx, res.Secret.ID, client.SecretUpdateOptions{
				Version: res.Secret.Meta.Version,
				Spec:    secretSpec,
			})
			if err != nil {
				return fmt.Errorf("failed to update secret %s: %w", secretSpec.Name, err)
			}
		case errdefs.IsNotFound(err):
			// secret does not exist, then we create a new one.
			_, _ = fmt.Fprintln(dockerCLI.Out(), "Creating secret", secretSpec.Name)
			_, err := apiClient.SecretCreate(ctx, client.SecretCreateOptions{
				Spec: secretSpec,
			})
			if err != nil {
				return fmt.Errorf("failed to create secret %s: %w", secretSpec.Name, err)
			}
		default:
			return err
		}
	}
	return nil
}

func createConfigs(ctx context.Context, dockerCLI command.Cli, configs []swarm.ConfigSpec) error {
	apiClient := dockerCLI.Client()

	for _, configSpec := range configs {
		res, err := apiClient.ConfigInspect(ctx, configSpec.Name, client.ConfigInspectOptions{})
		switch {
		case err == nil:
			// config already exists, then we update that
			_, err := apiClient.ConfigUpdate(ctx, res.Config.ID, client.ConfigUpdateOptions{
				Version: res.Config.Meta.Version,
				Spec:    configSpec,
			})
			if err != nil {
				return fmt.Errorf("failed to update config %s: %w", configSpec.Name, err)
			}
		case errdefs.IsNotFound(err):
			// config does not exist, then we create a new one.
			_, _ = fmt.Fprintln(dockerCLI.Out(), "Creating config", configSpec.Name)
			_, err := apiClient.ConfigCreate(ctx, client.ConfigCreateOptions{
				Spec: configSpec,
			})
			if err != nil {
				return fmt.Errorf("failed to create config %s: %w", configSpec.Name, err)
			}
		default:
			return err
		}
	}
	return nil
}

func createNetworks(ctx context.Context, dockerCLI command.Cli, namespace convert.Namespace, networks map[string]client.NetworkCreateOptions) error {
	apiClient := dockerCLI.Client()

	existingNetworks, err := getStackNetworks(ctx, apiClient, namespace.Name())
	if err != nil {
		return err
	}

	existingNetworkMap := make(map[string]network.Summary)
	for _, nw := range existingNetworks.Items {
		existingNetworkMap[nw.Name] = nw
	}

	for name, createOpts := range networks {
		if _, exists := existingNetworkMap[name]; exists {
			continue
		}

		if createOpts.Driver == "" {
			createOpts.Driver = defaultNetworkDriver
		}

		_, _ = fmt.Fprintln(dockerCLI.Out(), "Creating network", name)
		if _, err := apiClient.NetworkCreate(ctx, name, createOpts); err != nil {
			return fmt.Errorf("failed to create network %s: %w", name, err)
		}
	}
	return nil
}

func deployServices(ctx context.Context, dockerCLI command.Cli, services map[string]swarm.ServiceSpec, namespace convert.Namespace, sendAuth bool, resolveImage string) ([]string, error) {
	apiClient := dockerCLI.Client()
	out := dockerCLI.Out()

	existingServices, err := getStackServices(ctx, apiClient, namespace.Name())
	if err != nil {
		return nil, err
	}

	existingServiceMap := make(map[string]swarm.Service)
	for _, svc := range existingServices.Items {
		existingServiceMap[svc.Spec.Name] = svc
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
			encodedAuth, err = command.RetrieveAuthTokenFromImage(dockerCLI.ConfigFile(), image)
			if err != nil {
				return nil, err
			}
		}

		if svc, exists := existingServiceMap[name]; exists {
			_, _ = fmt.Fprintf(out, "Updating service %s (id: %s)\n", name, svc.ID)

			updateOpts := client.ServiceUpdateOptions{
				Version:             svc.Version,
				EncodedRegistryAuth: encodedAuth,
			}

			switch resolveImage {
			case resolveImageAlways:
				// image should be updated by the server using QueryRegistry
				updateOpts.QueryRegistry = true
			case resolveImageChanged:
				if image != svc.Spec.Labels[convert.LabelImage] {
					// Query the registry to resolve digest for the updated image
					updateOpts.QueryRegistry = true
				} else {
					// image has not changed; update the serviceSpec with the
					// existing information that was set by QueryRegistry on the
					// previous deploy. Otherwise this will trigger an incorrect
					// service update.
					serviceSpec.TaskTemplate.ContainerSpec.Image = svc.Spec.TaskTemplate.ContainerSpec.Image
				}
			default:
				if image == svc.Spec.Labels[convert.LabelImage] {
					// image has not changed; update the serviceSpec with the
					// existing information that was set by QueryRegistry on the
					// previous deploy. Otherwise this will trigger an incorrect
					// service update.
					serviceSpec.TaskTemplate.ContainerSpec.Image = svc.Spec.TaskTemplate.ContainerSpec.Image
				}
			}

			// Stack deploy does not have a `--force` option. Preserve existing
			// ForceUpdate value so that tasks are not re-deployed if not updated.
			// TODO move this to API client?
			serviceSpec.TaskTemplate.ForceUpdate = svc.Spec.TaskTemplate.ForceUpdate

			updateOpts.Spec = serviceSpec
			response, err := apiClient.ServiceUpdate(ctx, svc.ID, updateOpts)
			if err != nil {
				return nil, fmt.Errorf("failed to update service %s: %w", name, err)
			}

			for _, warning := range response.Warnings {
				_, _ = fmt.Fprintln(dockerCLI.Err(), warning)
			}

			serviceIDs = append(serviceIDs, svc.ID)
		} else {
			_, _ = fmt.Fprintln(out, "Creating service", name)

			// query registry if flag disabling it was not set
			queryRegistry := resolveImage == resolveImageAlways || resolveImage == resolveImageChanged

			response, err := apiClient.ServiceCreate(ctx, client.ServiceCreateOptions{
				Spec:                serviceSpec,
				EncodedRegistryAuth: encodedAuth,
				QueryRegistry:       queryRegistry,
			})
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
		if err := service.WaitOnService(ctx, dockerCli, serviceID, quiet); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", serviceID, err))
		}
	}
	return errors.Join(errs...)
}
