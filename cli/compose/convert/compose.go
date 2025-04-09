package convert

import (
	"os"
	"strings"

	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
)

const (
	// LabelNamespace is the label used to track stack resources
	LabelNamespace = "com.docker.stack.namespace"
)

// Namespace mangles names by prepending the name
type Namespace struct {
	name string
}

// Scope prepends the namespace to a name
func (n Namespace) Scope(name string) string {
	return n.name + "_" + name
}

// Descope returns the name without the namespace prefix
func (n Namespace) Descope(name string) string {
	return strings.TrimPrefix(name, n.name+"_")
}

// Name returns the name of the namespace
func (n Namespace) Name() string {
	return n.name
}

// NewNamespace returns a new Namespace for scoping of names
func NewNamespace(name string) Namespace {
	return Namespace{name: name}
}

// AddStackLabel returns labels with the namespace label added
func AddStackLabel(namespace Namespace, labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[LabelNamespace] = namespace.name
	return labels
}

type networkMap map[string]composetypes.NetworkConfig

// Networks from the compose-file type to the engine API type
func Networks(namespace Namespace, networks networkMap, servicesNetworks map[string]struct{}) (map[string]network.CreateOptions, []string) {
	if networks == nil {
		networks = make(map[string]composetypes.NetworkConfig)
	}

	externalNetworks := []string{}
	result := make(map[string]network.CreateOptions)
	for internalName := range servicesNetworks {
		nw := networks[internalName]
		if nw.External.External {
			externalNetworks = append(externalNetworks, nw.Name)
			continue
		}

		createOpts := network.CreateOptions{
			Labels:     AddStackLabel(namespace, nw.Labels),
			Driver:     nw.Driver,
			Options:    nw.DriverOpts,
			Internal:   nw.Internal,
			Attachable: nw.Attachable,
		}

		if nw.Ipam.Driver != "" || len(nw.Ipam.Config) > 0 {
			createOpts.IPAM = &network.IPAM{}
		}

		if nw.Ipam.Driver != "" {
			createOpts.IPAM.Driver = nw.Ipam.Driver
		}
		for _, ipamConfig := range nw.Ipam.Config {
			config := network.IPAMConfig{
				Subnet: ipamConfig.Subnet,
			}
			createOpts.IPAM.Config = append(createOpts.IPAM.Config, config)
		}

		networkName := namespace.Scope(internalName)
		if nw.Name != "" {
			networkName = nw.Name
		}
		result[networkName] = createOpts
	}

	return result, externalNetworks
}

// Secrets converts secrets from the Compose type to the engine API type
func Secrets(namespace Namespace, secrets map[string]composetypes.SecretConfig) ([]swarm.SecretSpec, error) {
	result := []swarm.SecretSpec{}
	for name, secret := range secrets {
		if secret.External.External {
			continue
		}

		var obj swarmFileObject
		var err error
		if secret.Driver != "" {
			obj = driverObjectConfig(namespace, name, composetypes.FileObjectConfig(secret))
		} else {
			obj, err = fileObjectConfig(namespace, name, composetypes.FileObjectConfig(secret))
		}
		if err != nil {
			return nil, err
		}
		spec := swarm.SecretSpec{Annotations: obj.Annotations, Data: obj.Data}
		if secret.Driver != "" {
			spec.Driver = &swarm.Driver{
				Name:    secret.Driver,
				Options: secret.DriverOpts,
			}
		}
		if secret.TemplateDriver != "" {
			spec.Templating = &swarm.Driver{
				Name: secret.TemplateDriver,
			}
		}
		result = append(result, spec)
	}
	return result, nil
}

// Configs converts config objects from the Compose type to the engine API type
func Configs(namespace Namespace, configs map[string]composetypes.ConfigObjConfig) ([]swarm.ConfigSpec, error) {
	result := []swarm.ConfigSpec{}
	for name, config := range configs {
		if config.External.External {
			continue
		}

		obj, err := fileObjectConfig(namespace, name, composetypes.FileObjectConfig(config))
		if err != nil {
			return nil, err
		}
		spec := swarm.ConfigSpec{Annotations: obj.Annotations, Data: obj.Data}
		if config.TemplateDriver != "" {
			spec.Templating = &swarm.Driver{
				Name: config.TemplateDriver,
			}
		}
		result = append(result, spec)
	}
	return result, nil
}

type swarmFileObject struct {
	Annotations swarm.Annotations
	Data        []byte
}

func driverObjectConfig(namespace Namespace, name string, obj composetypes.FileObjectConfig) swarmFileObject {
	if obj.Name != "" {
		name = obj.Name
	} else {
		name = namespace.Scope(name)
	}

	return swarmFileObject{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: AddStackLabel(namespace, obj.Labels),
		},
		Data: []byte{},
	}
}

func fileObjectConfig(namespace Namespace, name string, obj composetypes.FileObjectConfig) (swarmFileObject, error) {
	data, err := os.ReadFile(obj.File)
	if err != nil {
		return swarmFileObject{}, err
	}

	if obj.Name != "" {
		name = obj.Name
	} else {
		name = namespace.Scope(name)
	}

	return swarmFileObject{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: AddStackLabel(namespace, obj.Labels),
		},
		Data: data,
	}, nil
}
