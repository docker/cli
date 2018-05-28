package swarm

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/loader"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/compose/convert"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/swarm"
	apiclient "github.com/docker/docker/client"
	dockerclient "github.com/docker/docker/client"
	"github.com/pkg/errors"
)

func deployCompose(ctx context.Context, dockerCli command.Cli, opts options.Deploy) error {
	if err := validateStackName(opts.Namespace); err != nil {
		return err
	}
	config, err := loader.LoadComposefile(dockerCli, opts)
	if err != nil {
		return err
	}

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

	services, err := convert.Services(namespace, config, dockerCli.Client())
	if err != nil {
		return err
	}
	return deployServices(ctx, dockerCli, services, namespace, opts.SendRegistryAuth, opts.ResolveImage)
}

func getServicesDeclaredNetworks(serviceConfigs []composetypes.ServiceConfig) map[string]struct{} {
	serviceNetworks := map[string]struct{}{}
	for _, serviceConfig := range serviceConfigs {
		if len(serviceConfig.Networks) == 0 {
			serviceNetworks["default"] = struct{}{}
			continue
		}
		for network := range serviceConfig.Networks {
			serviceNetworks[network] = struct{}{}
		}
	}
	return serviceNetworks
}

func validateExternalNetworks(
	ctx context.Context,
	client dockerclient.NetworkAPIClient,
	externalNetworks []string,
) error {
	for _, networkName := range externalNetworks {
		if !container.NetworkMode(networkName).IsUserDefined() {
			// Networks that are not user defined always exist on all nodes as
			// local-scoped networks, so there's no need to inspect them.
			continue
		}
		network, err := client.NetworkInspect(ctx, networkName, types.NetworkInspectOptions{})
		switch {
		case dockerclient.IsErrNotFound(err):
			return errors.Errorf("network %q is declared as external, but could not be found. You need to create a swarm-scoped network before the stack is deployed", networkName)
		case err != nil:
			return err
		case network.Scope != "swarm":
			return errors.Errorf("network %q is declared as external, but it is not in the right scope: %q instead of \"swarm\"", networkName, network.Scope)
		}
	}
	return nil
}

func createSecrets(
	ctx context.Context,
	dockerCli command.Cli,
	secrets []swarm.SecretSpec,
) error {
	client := dockerCli.Client()

	for _, secretSpec := range secrets {
		secret, _, err := client.SecretInspectWithRaw(ctx, secretSpec.Name)
		switch {
		case err == nil:
			// secret already exists, then we update that
			if err := client.SecretUpdate(ctx, secret.ID, secret.Meta.Version, secretSpec); err != nil {
				return errors.Wrapf(err, "failed to update secret %s", secretSpec.Name)
			}
		case apiclient.IsErrNotFound(err):
			// secret does not exist, then we create a new one.
			fmt.Fprintf(dockerCli.Out(), "Creating secret %s\n", secretSpec.Name)
			if _, err := client.SecretCreate(ctx, secretSpec); err != nil {
				return errors.Wrapf(err, "failed to create secret %s", secretSpec.Name)
			}
		default:
			return err
		}
	}
	return nil
}

func createConfigs(
	ctx context.Context,
	dockerCli command.Cli,
	configs []swarm.ConfigSpec,
) error {
	client := dockerCli.Client()

	for _, configSpec := range configs {
		config, _, err := client.ConfigInspectWithRaw(ctx, configSpec.Name)
		switch {
		case err == nil:
			// config already exists, then we update that
			if err := client.ConfigUpdate(ctx, config.ID, config.Meta.Version, configSpec); err != nil {
				return errors.Wrapf(err, "failed to update config %s", configSpec.Name)
			}
		case apiclient.IsErrNotFound(err):
			// config does not exist, then we create a new one.
			fmt.Fprintf(dockerCli.Out(), "Creating config %s\n", configSpec.Name)
			if _, err := client.ConfigCreate(ctx, configSpec); err != nil {
				return errors.Wrapf(err, "failed to create config %s", configSpec.Name)
			}
		default:
			return err
		}
	}
	return nil
}

func createNetworks(
	ctx context.Context,
	dockerCli command.Cli,
	namespace convert.Namespace,
	networks map[string]types.NetworkCreate,
) error {
	client := dockerCli.Client()

	existingNetworks, err := getStackNetworks(ctx, client, namespace.Name())
	if err != nil {
		return err
	}

	existingNetworkMap := make(map[string]types.NetworkResource)
	for _, network := range existingNetworks {
		existingNetworkMap[network.Name] = network
	}

	for name, createOpts := range networks {
		if _, exists := existingNetworkMap[name]; exists {
			continue
		}

		if createOpts.Driver == "" {
			createOpts.Driver = defaultNetworkDriver
		}

		fmt.Fprintf(dockerCli.Out(), "Creating network %s\n", name)
		if _, err := client.NetworkCreate(ctx, name, createOpts); err != nil {
			return errors.Wrapf(err, "failed to create network %s", name)
		}
	}
	return nil
}

func deployServices(
	ctx context.Context,
	dockerCli command.Cli,
	services map[string]swarm.ServiceSpec,
	namespace convert.Namespace,
	sendAuth bool,
	resolveImage string,
) error {
	apiClient := dockerCli.Client()
	out := dockerCli.Out()

	existingServices, err := getStackServices(ctx, apiClient, namespace.Name())
	if err != nil {
		return err
	}

	existingServiceMap := make(map[string]swarm.Service)
	for _, service := range existingServices {
		existingServiceMap[service.Spec.Name] = service
	}

	for internalName, serviceSpec := range services {
		name := namespace.Scope(internalName)

		encodedAuth := ""
		image := serviceSpec.TaskTemplate.ContainerSpec.Image
		if sendAuth {
			// Retrieve encoded auth token from the image reference
			encodedAuth, err = command.RetrieveAuthTokenFromImage(ctx, dockerCli, image)
			if err != nil {
				return err
			}
		}

		if service, exists := existingServiceMap[name]; exists {
			fmt.Fprintf(out, "Updating service %s (id: %s)\n", name, service.ID)

			updateOpts := types.ServiceUpdateOptions{EncodedRegistryAuth: encodedAuth}

			switch {
			case resolveImage == ResolveImageAlways || (resolveImage == ResolveImageChanged && image != service.Spec.Labels[convert.LabelImage]):
				// image should be updated by the server using QueryRegistry
				updateOpts.QueryRegistry = true
			case image == service.Spec.Labels[convert.LabelImage]:
				// image has not changed; update the serviceSpec with the
				// existing information that was set by QueryRegistry on the
				// previous deploy. Otherwise this will trigger an incorrect
				// service update.
				serviceSpec.TaskTemplate.ContainerSpec.Image = service.Spec.TaskTemplate.ContainerSpec.Image
			}

			// Stack deploy does not have a `--force` option. Preserve existing ForceUpdate
			// value so that tasks are not re-deployed if not updated.
			// TODO move this to API client?
			serviceSpec.TaskTemplate.ForceUpdate = service.Spec.TaskTemplate.ForceUpdate

			response, err := apiClient.ServiceUpdate(
				ctx,
				service.ID,
				service.Version,
				serviceSpec,
				updateOpts,
			)
			if err != nil {
				return errors.Wrapf(err, "failed to update service %s", name)
			}

			for _, warning := range response.Warnings {
				fmt.Fprintln(dockerCli.Err(), warning)
			}
		} else {
			fmt.Fprintf(out, "Creating service %s\n", name)

			createOpts := types.ServiceCreateOptions{EncodedRegistryAuth: encodedAuth}

			// query registry if flag disabling it was not set
			if resolveImage == ResolveImageAlways || resolveImage == ResolveImageChanged {
				createOpts.QueryRegistry = true
			}

			if _, err := apiClient.ServiceCreate(ctx, serviceSpec, createOpts); err != nil {
				return errors.Wrapf(err, "failed to create service %s", name)
			}
		}
	}
	return nil
}
