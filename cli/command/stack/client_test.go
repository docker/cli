package stack

import (
	"context"
	"strings"

	"github.com/docker/cli/cli/compose/convert"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

type fakeClient struct {
	client.Client

	services []string
	networks []string
	secrets  []string
	configs  []string

	removedServices []string
	removedNetworks []string
	removedSecrets  []string
	removedConfigs  []string

	serviceListFunc   func(options client.ServiceListOptions) (client.ServiceListResult, error)
	networkListFunc   func(options client.NetworkListOptions) (client.NetworkListResult, error)
	secretListFunc    func(options client.SecretListOptions) (client.SecretListResult, error)
	configListFunc    func(options client.ConfigListOptions) (client.ConfigListResult, error)
	nodeListFunc      func(options client.NodeListOptions) (client.NodeListResult, error)
	taskListFunc      func(options client.TaskListOptions) (client.TaskListResult, error)
	nodeInspectFunc   func(ref string) (client.NodeInspectResult, error)
	serviceUpdateFunc func(serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error)
	serviceRemoveFunc func(serviceID string) (client.ServiceRemoveResult, error)
	networkRemoveFunc func(networkID string) error
	secretRemoveFunc  func(secretID string) (client.SecretRemoveResult, error)
	configRemoveFunc  func(configID string) (client.ConfigRemoveResult, error)
}

func (*fakeClient) ServerVersion(context.Context, client.ServerVersionOptions) (client.ServerVersionResult, error) {
	return client.ServerVersionResult{
		APIVersion: client.MaxAPIVersion,
	}, nil
}

func (*fakeClient) ClientVersion() string {
	return client.MaxAPIVersion
}

func (cli *fakeClient) ServiceList(_ context.Context, options client.ServiceListOptions) (client.ServiceListResult, error) {
	if cli.serviceListFunc != nil {
		return cli.serviceListFunc(options)
	}

	namespace := namespaceFromFilters(options.Filters)
	servicesList := client.ServiceListResult{}
	for _, name := range cli.services {
		if belongToNamespace(name, namespace) {
			servicesList.Items = append(servicesList.Items, serviceFromName(name))
		}
	}
	return servicesList, nil
}

func (cli *fakeClient) NetworkList(_ context.Context, options client.NetworkListOptions) (client.NetworkListResult, error) {
	if cli.networkListFunc != nil {
		return cli.networkListFunc(options)
	}

	namespace := namespaceFromFilters(options.Filters)
	networksList := client.NetworkListResult{}
	for _, name := range cli.networks {
		if belongToNamespace(name, namespace) {
			networksList.Items = append(networksList.Items, networkFromName(name))
		}
	}
	return networksList, nil
}

func (cli *fakeClient) SecretList(_ context.Context, options client.SecretListOptions) (client.SecretListResult, error) {
	if cli.secretListFunc != nil {
		return cli.secretListFunc(options)
	}

	namespace := namespaceFromFilters(options.Filters)
	secretsList := client.SecretListResult{}
	for _, name := range cli.secrets {
		if belongToNamespace(name, namespace) {
			secretsList.Items = append(secretsList.Items, secretFromName(name))
		}
	}
	return secretsList, nil
}

func (cli *fakeClient) ConfigList(_ context.Context, options client.ConfigListOptions) (client.ConfigListResult, error) {
	if cli.configListFunc != nil {
		return cli.configListFunc(options)
	}

	namespace := namespaceFromFilters(options.Filters)
	configsList := client.ConfigListResult{}
	for _, name := range cli.configs {
		if belongToNamespace(name, namespace) {
			configsList.Items = append(configsList.Items, configFromName(name))
		}
	}
	return configsList, nil
}

func (cli *fakeClient) TaskList(_ context.Context, options client.TaskListOptions) (client.TaskListResult, error) {
	if cli.taskListFunc != nil {
		return cli.taskListFunc(options)
	}
	return client.TaskListResult{}, nil
}

func (cli *fakeClient) NodeList(_ context.Context, options client.NodeListOptions) (client.NodeListResult, error) {
	if cli.nodeListFunc != nil {
		return cli.nodeListFunc(options)
	}
	return client.NodeListResult{}, nil
}

func (cli *fakeClient) NodeInspect(_ context.Context, ref string, _ client.NodeInspectOptions) (client.NodeInspectResult, error) {
	if cli.nodeInspectFunc != nil {
		return cli.nodeInspectFunc(ref)
	}
	return client.NodeInspectResult{}, nil
}

func (cli *fakeClient) ServiceUpdate(_ context.Context, serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error) {
	if cli.serviceUpdateFunc != nil {
		return cli.serviceUpdateFunc(serviceID, options)
	}

	return client.ServiceUpdateResult{}, nil
}

func (cli *fakeClient) ServiceRemove(_ context.Context, serviceID string, _ client.ServiceRemoveOptions) (client.ServiceRemoveResult, error) {
	if cli.serviceRemoveFunc != nil {
		return cli.serviceRemoveFunc(serviceID)
	}

	cli.removedServices = append(cli.removedServices, serviceID)
	return client.ServiceRemoveResult{}, nil
}

func (cli *fakeClient) NetworkRemove(_ context.Context, networkID string, _ client.NetworkRemoveOptions) (client.NetworkRemoveResult, error) {
	if cli.networkRemoveFunc != nil {
		return client.NetworkRemoveResult{}, cli.networkRemoveFunc(networkID)
	}

	cli.removedNetworks = append(cli.removedNetworks, networkID)
	return client.NetworkRemoveResult{}, nil
}

func (cli *fakeClient) SecretRemove(_ context.Context, secretID string, _ client.SecretRemoveOptions) (client.SecretRemoveResult, error) {
	if cli.secretRemoveFunc != nil {
		return cli.secretRemoveFunc(secretID)
	}

	cli.removedSecrets = append(cli.removedSecrets, secretID)
	return client.SecretRemoveResult{}, nil
}

func (cli *fakeClient) ConfigRemove(_ context.Context, configID string, _ client.ConfigRemoveOptions) (client.ConfigRemoveResult, error) {
	if cli.configRemoveFunc != nil {
		return cli.configRemoveFunc(configID)
	}

	cli.removedConfigs = append(cli.removedConfigs, configID)
	return client.ConfigRemoveResult{}, nil
}

func (*fakeClient) ServiceInspect(_ context.Context, serviceID string, _ client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
	return client.ServiceInspectResult{
		Service: swarm.Service{
			ID: serviceID,
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: serviceID,
				},
			},
		},
	}, nil
}

func serviceFromName(name string) swarm.Service {
	return swarm.Service{
		ID: "ID-" + name,
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{Name: name},
		},
	}
}

func networkFromName(name string) network.Summary {
	return network.Summary{
		Network: network.Network{
			ID:   "ID-" + name,
			Name: name,
		},
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

func namespaceFromFilters(fltrs client.Filters) string {
	// FIXME(thaJeztah): more elegant way for this? Should we have a utility for this?
	var label string
	for fltr := range fltrs["label"] {
		label = fltr
		break
	}
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
