package swarm

import (
	"context"
	"strings"

	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

type fakeClient struct {
	client.Client

	version string

	services []string
	networks []string
	secrets  []string
	configs  []string

	removedServices []string
	removedNetworks []string
	removedSecrets  []string
	removedConfigs  []string

	serviceListFunc    func(options types.ServiceListOptions) ([]swarm.Service, error)
	networkListFunc    func(options types.NetworkListOptions) ([]types.NetworkResource, error)
	secretListFunc     func(options types.SecretListOptions) ([]swarm.Secret, error)
	configListFunc     func(options types.ConfigListOptions) ([]swarm.Config, error)
	nodeListFunc       func(options types.NodeListOptions) ([]swarm.Node, error)
	taskListFunc       func(options types.TaskListOptions) ([]swarm.Task, error)
	nodeInspectWithRaw func(ref string) (swarm.Node, []byte, error)

	serviceUpdateFunc func(serviceID string, version swarm.Version, service swarm.ServiceSpec, options types.ServiceUpdateOptions) (types.ServiceUpdateResponse, error)

	serviceRemoveFunc func(serviceID string) error
	networkRemoveFunc func(networkID string) error
	secretRemoveFunc  func(secretID string) error
	configRemoveFunc  func(configID string) error
}

func (cli *fakeClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return types.Version{
		Version:    "docker-dev",
		APIVersion: api.DefaultVersion,
	}, nil
}

func (cli *fakeClient) ClientVersion() string {
	return cli.version
}

func (cli *fakeClient) ServiceList(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
	if cli.serviceListFunc != nil {
		return cli.serviceListFunc(options)
	}

	namespace := namespaceFromFilters(options.Filters)
	servicesList := []swarm.Service{}
	for _, name := range cli.services {
		if belongToNamespace(name, namespace) {
			servicesList = append(servicesList, serviceFromName(name))
		}
	}
	return servicesList, nil
}

func (cli *fakeClient) NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
	if cli.networkListFunc != nil {
		return cli.networkListFunc(options)
	}

	namespace := namespaceFromFilters(options.Filters)
	networksList := []types.NetworkResource{}
	for _, name := range cli.networks {
		if belongToNamespace(name, namespace) {
			networksList = append(networksList, networkFromName(name))
		}
	}
	return networksList, nil
}

func (cli *fakeClient) SecretList(ctx context.Context, options types.SecretListOptions) ([]swarm.Secret, error) {
	if cli.secretListFunc != nil {
		return cli.secretListFunc(options)
	}

	namespace := namespaceFromFilters(options.Filters)
	secretsList := []swarm.Secret{}
	for _, name := range cli.secrets {
		if belongToNamespace(name, namespace) {
			secretsList = append(secretsList, secretFromName(name))
		}
	}
	return secretsList, nil
}

func (cli *fakeClient) ConfigList(ctx context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
	if cli.configListFunc != nil {
		return cli.configListFunc(options)
	}

	namespace := namespaceFromFilters(options.Filters)
	configsList := []swarm.Config{}
	for _, name := range cli.configs {
		if belongToNamespace(name, namespace) {
			configsList = append(configsList, configFromName(name))
		}
	}
	return configsList, nil
}

func (cli *fakeClient) TaskList(ctx context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
	if cli.taskListFunc != nil {
		return cli.taskListFunc(options)
	}
	return []swarm.Task{}, nil
}

func (cli *fakeClient) NodeList(ctx context.Context, options types.NodeListOptions) ([]swarm.Node, error) {
	if cli.nodeListFunc != nil {
		return cli.nodeListFunc(options)
	}
	return []swarm.Node{}, nil
}

func (cli *fakeClient) NodeInspectWithRaw(ctx context.Context, ref string) (swarm.Node, []byte, error) {
	if cli.nodeInspectWithRaw != nil {
		return cli.nodeInspectWithRaw(ref)
	}
	return swarm.Node{}, nil, nil
}

func (cli *fakeClient) ServiceUpdate(ctx context.Context, serviceID string, version swarm.Version, service swarm.ServiceSpec, options types.ServiceUpdateOptions) (types.ServiceUpdateResponse, error) {
	if cli.serviceUpdateFunc != nil {
		return cli.serviceUpdateFunc(serviceID, version, service, options)
	}

	return types.ServiceUpdateResponse{}, nil
}

func (cli *fakeClient) ServiceRemove(ctx context.Context, serviceID string) error {
	if cli.serviceRemoveFunc != nil {
		return cli.serviceRemoveFunc(serviceID)
	}

	cli.removedServices = append(cli.removedServices, serviceID)
	return nil
}

func (cli *fakeClient) NetworkRemove(ctx context.Context, networkID string) error {
	if cli.networkRemoveFunc != nil {
		return cli.networkRemoveFunc(networkID)
	}

	cli.removedNetworks = append(cli.removedNetworks, networkID)
	return nil
}

func (cli *fakeClient) SecretRemove(ctx context.Context, secretID string) error {
	if cli.secretRemoveFunc != nil {
		return cli.secretRemoveFunc(secretID)
	}

	cli.removedSecrets = append(cli.removedSecrets, secretID)
	return nil
}

func (cli *fakeClient) ConfigRemove(ctx context.Context, configID string) error {
	if cli.configRemoveFunc != nil {
		return cli.configRemoveFunc(configID)
	}

	cli.removedConfigs = append(cli.removedConfigs, configID)
	return nil
}

func serviceFromName(name string) swarm.Service {
	return swarm.Service{
		ID: "ID-" + name,
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{Name: name},
		},
	}
}

func networkFromName(name string) types.NetworkResource {
	return types.NetworkResource{
		ID:   "ID-" + name,
		Name: name,
	}
}

func secretFromName(name string) swarm.Secret {
	return swarm.Secret{
		ID: "ID-" + name,
		Spec: swarm.SecretSpec{
			Annotations: swarm.Annotations{Name: name},
		},
	}
}

func configFromName(name string) swarm.Config {
	return swarm.Config{
		ID: "ID-" + name,
		Spec: swarm.ConfigSpec{
			Annotations: swarm.Annotations{Name: name},
		},
	}
}

func namespaceFromFilters(filters filters.Args) string {
	label := filters.Get("label")[0]
	return strings.TrimPrefix(label, convert.LabelNamespace+"=")
}

func belongToNamespace(id, namespace string) bool {
	return strings.HasPrefix(id, namespace+"_")
}

func objectName(namespace, name string) string {
	return namespace + "_" + name
}

func objectID(name string) string {
	return "ID-" + name
}

func buildObjectIDs(objectNames []string) []string {
	IDs := make([]string, len(objectNames))
	for i, name := range objectNames {
		IDs[i] = objectID(name)
	}
	return IDs
}
