package convert

import (
	"fmt"
	"net/netip"
	"os"
	"strings"

	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
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
	return addStackLabel(namespace, labels)
}

// addStackLabel returns labels with the namespace label added
func addStackLabel(namespace Namespace, labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[LabelNamespace] = namespace.name
	return labels
}

type networkMap map[string]composetypes.NetworkConfig

// Networks from the compose-file type to the engine API type
func Networks(namespace Namespace, networks networkMap, servicesNetworks map[string]struct{}) (map[string]client.NetworkCreateOptions, []string) {
	if networks == nil {
		networks = make(map[string]composetypes.NetworkConfig)
	}

	externalNetworks := []string{}
	result := make(map[string]client.NetworkCreateOptions)
	for internalName := range servicesNetworks {
		nw := networks[internalName]
		if nw.External.External {
			externalNetworks = append(externalNetworks, nw.Name)
			continue
		}

		createOpts := client.NetworkCreateOptions{
			Labels:     addStackLabel(namespace, nw.Labels),
			Driver:     nw.Driver,
			Options:    nw.DriverOpts,
			Internal:   nw.Internal,
			Attachable: nw.Attachable,
		}

		if nw.Ipam.Driver != "" || len(nw.Ipam.Config) > 0 {
			createOpts.IPAM = &network.IPAM{
				Driver: nw.Ipam.Driver,
			}
			for _, ipamConfig := range nw.Ipam.Config {
				sn, _ := parsePrefixOrAddr(ipamConfig.Subnet) // TODO(thaJeztah): change Subnet field to netip.Prefix (but this would break "address only" formats.
				createOpts.IPAM.Config = append(createOpts.IPAM.Config, network.IPAMConfig{
					Subnet: sn,
				})
			}
		}

		networkName := nw.Name
		if nw.Name == "" {
			networkName = namespace.Scope(internalName)
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
			Labels: addStackLabel(namespace, obj.Labels),
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
			Labels: addStackLabel(namespace, obj.Labels),
		},
		Data: data,
	}, nil
}

// parsePrefixOrAddr parses s as a subnet in CIDR notation (e.g. "10.0.0.0/24").
// If s does not include a prefix length, it is interpreted as a single-address
// subnet using the full address width (/32 for IPv4 or /128 for IPv6).
//
// It returns the resulting netip.Prefix or an error if the input is invalid.
func parsePrefixOrAddr(s string) (netip.Prefix, error) {
	pfx, err := netip.ParsePrefix(s)
	if err != nil {
		addr, err := netip.ParseAddr(s)
		if err != nil {
			return netip.Prefix{}, fmt.Errorf("invalid address: %w", err)
		}
		pfx = netip.PrefixFrom(addr, addr.BitLen())
	}
	return pfx, nil
}
