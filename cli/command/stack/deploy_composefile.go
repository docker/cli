package stack

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/cli/cli/compose/loader"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/swarm"
	apiclient "github.com/docker/docker/client"
	dockerclient "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

func deployCompose(ctx context.Context, dockerCli command.Cli, opts deployOptions) error {
	configDetails, err := getConfigDetails(opts.composefile, dockerCli.In())
	if err != nil {
		return err
	}

	config, err := loader.Load(configDetails)
	if err != nil {
		if fpe, ok := err.(*loader.ForbiddenPropertiesError); ok {
			return errors.Errorf("Compose file contains unsupported options:\n\n%s\n",
				propertyWarnings(fpe.Properties))
		}

		return err
	}

	unsupportedProperties := loader.GetUnsupportedProperties(configDetails)
	if len(unsupportedProperties) > 0 {
		fmt.Fprintf(dockerCli.Err(), "Ignoring unsupported options: %s\n\n",
			strings.Join(unsupportedProperties, ", "))
	}

	deprecatedProperties := loader.GetDeprecatedProperties(configDetails)
	if len(deprecatedProperties) > 0 {
		fmt.Fprintf(dockerCli.Err(), "Ignoring deprecated options:\n\n%s\n\n",
			propertyWarnings(deprecatedProperties))
	}

	if err := checkDaemonIsSwarmManager(ctx, dockerCli); err != nil {
		return err
	}

	namespace := convert.NewNamespace(opts.namespace)

	if opts.prune {
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
	return deployServices(ctx, dockerCli, services, namespace, opts.sendRegistryAuth, opts.resolveImage)
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

func propertyWarnings(properties map[string]string) string {
	var msgs []string
	for name, description := range properties {
		msgs = append(msgs, fmt.Sprintf("%s: %s", name, description))
	}
	sort.Strings(msgs)
	return strings.Join(msgs, "\n\n")
}

func getConfigDetails(composefile string, stdin io.Reader) (composetypes.ConfigDetails, error) {
	var details composetypes.ConfigDetails

	if composefile == "-" {
		workingDir, err := os.Getwd()
		if err != nil {
			return details, err
		}
		details.WorkingDir = workingDir
	} else {
		absPath, err := filepath.Abs(composefile)
		if err != nil {
			return details, err
		}
		details.WorkingDir = filepath.Dir(absPath)
	}

	configFile, err := getConfigFile(composefile, stdin)
	if err != nil {
		return details, err
	}
	// TODO: support multiple files
	details.ConfigFiles = []composetypes.ConfigFile{*configFile}
	details.Environment, err = buildEnvironment(os.Environ())
	return details, err
}

func buildEnvironment(env []string) (map[string]string, error) {
	result := make(map[string]string, len(env))
	for _, s := range env {
		// if value is empty, s is like "K=", not "K".
		if !strings.Contains(s, "=") {
			return result, errors.Errorf("unexpected environment %q", s)
		}
		kv := strings.SplitN(s, "=", 2)
		result[kv[0]] = kv[1]
	}
	return result, nil
}

func getConfigFile(filename string, stdin io.Reader) (*composetypes.ConfigFile, error) {
	var bytes []byte
	var err error

	if filename == "-" {
		bytes, err = ioutil.ReadAll(stdin)
	} else {
		bytes, err = ioutil.ReadFile(filename)
	}
	if err != nil {
		return nil, err
	}

	config, err := loader.ParseYAML(bytes)
	if err != nil {
		return nil, err
	}

	return &composetypes.ConfigFile{
		Filename: filename,
		Config:   config,
	}, nil
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

	for internalName, createOpts := range networks {
		name := namespace.Scope(internalName)
		if _, exists := existingNetworkMap[name]; exists {
			continue
		}

		if createOpts.Driver == "" {
			createOpts.Driver = defaultNetworkDriver
		}

		fmt.Fprintf(dockerCli.Out(), "Creating network %s\n", name)
		if _, err := client.NetworkCreate(ctx, name, createOpts); err != nil {
			return errors.Wrapf(err, "failed to create network %s", internalName)
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

	existingServices, err := getServices(ctx, apiClient, namespace.Name())
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
			case resolveImage == resolveImageAlways || (resolveImage == resolveImageChanged && image != service.Spec.Labels[convert.LabelImage]):
				// image should be updated by the server using QueryRegistry
				updateOpts.QueryRegistry = true
			case image == service.Spec.Labels[convert.LabelImage]:
				// image has not changed; update the serviceSpec with the
				// existing information that was set by QueryRegistry on the
				// previous deploy. Otherwise this will trigger an incorrect
				// service update.
				serviceSpec.TaskTemplate.ContainerSpec.Image = service.Spec.TaskTemplate.ContainerSpec.Image
			}
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
			if resolveImage == resolveImageAlways || resolveImage == resolveImageChanged {
				createOpts.QueryRegistry = true
			}

			if _, err := apiClient.ServiceCreate(ctx, serviceSpec, createOpts); err != nil {
				return errors.Wrapf(err, "failed to create service %s", name)
			}
		}
	}
	return nil
}
