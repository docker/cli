package convert

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	servicecli "github.com/docker/cli/cli/command/service"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

const (
	defaultNetwork = "default"
	// LabelImage is the label used to store image name provided in the compose file
	LabelImage = "com.docker.stack.image"
)

// Services from compose-file types to engine API types
func Services(
	namespace Namespace,
	config *composetypes.Config,
	client client.CommonAPIClient,
) (map[string]swarm.ServiceSpec, error) {
	result := make(map[string]swarm.ServiceSpec)

	services := config.Services
	volumes := config.Volumes
	networks := config.Networks

	for _, service := range services {
		secrets, err := convertServiceSecrets(client, namespace, service.Secrets, config.Secrets)
		if err != nil {
			return nil, errors.Wrapf(err, "service %s", service.Name)
		}
		configs, err := convertServiceConfigObjs(client, namespace, service, config.Configs)
		if err != nil {
			return nil, errors.Wrapf(err, "service %s", service.Name)
		}

		serviceSpec, err := Service(client.ClientVersion(), namespace, service, networks, volumes, secrets, configs)
		if err != nil {
			return nil, errors.Wrapf(err, "service %s", service.Name)
		}
		result[service.Name] = serviceSpec
	}

	return result, nil
}

// Service converts a ServiceConfig into a swarm ServiceSpec
func Service(
	apiVersion string,
	namespace Namespace,
	service composetypes.ServiceConfig,
	networkConfigs map[string]composetypes.NetworkConfig,
	volumes map[string]composetypes.VolumeConfig,
	secrets []*swarm.SecretReference,
	configs []*swarm.ConfigReference,
) (swarm.ServiceSpec, error) {
	name := namespace.Scope(service.Name)

	endpoint, err := convertEndpointSpec(service.Deploy.EndpointMode, service.Ports)
	if err != nil {
		return swarm.ServiceSpec{}, err
	}

	mode, err := convertDeployMode(service.Deploy.Mode, service.Deploy.Replicas)
	if err != nil {
		return swarm.ServiceSpec{}, err
	}

	mounts, err := Volumes(service.Volumes, volumes, namespace)
	if err != nil {
		return swarm.ServiceSpec{}, err
	}

	resources, err := convertResources(service.Deploy.Resources)
	if err != nil {
		return swarm.ServiceSpec{}, err
	}

	restartPolicy, err := convertRestartPolicy(
		service.Restart, service.Deploy.RestartPolicy)
	if err != nil {
		return swarm.ServiceSpec{}, err
	}

	healthcheck, err := convertHealthcheck(service.HealthCheck)
	if err != nil {
		return swarm.ServiceSpec{}, err
	}

	networks, err := convertServiceNetworks(service.Networks, networkConfigs, namespace, service.Name)
	if err != nil {
		return swarm.ServiceSpec{}, err
	}

	dnsConfig, err := convertDNSConfig(service.DNS, service.DNSSearch)
	if err != nil {
		return swarm.ServiceSpec{}, err
	}

	var privileges swarm.Privileges
	privileges.CredentialSpec, err = convertCredentialSpec(
		namespace, service.CredentialSpec, configs,
	)
	if err != nil {
		return swarm.ServiceSpec{}, err
	}

	var logDriver *swarm.Driver
	if service.Logging != nil {
		logDriver = &swarm.Driver{
			Name:    service.Logging.Driver,
			Options: service.Logging.Options,
		}
	}

	serviceSpec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: AddStackLabel(namespace, service.Deploy.Labels),
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:           service.Image,
				Command:         service.Entrypoint,
				Args:            service.Command,
				Hostname:        service.Hostname,
				Hosts:           convertExtraHosts(service.ExtraHosts),
				DNSConfig:       dnsConfig,
				Healthcheck:     healthcheck,
				Env:             sortStrings(convertEnvironment(service.Environment)),
				Labels:          AddStackLabel(namespace, service.Labels),
				Dir:             service.WorkingDir,
				User:            service.User,
				Mounts:          mounts,
				StopGracePeriod: composetypes.ConvertDurationPtr(service.StopGracePeriod),
				StopSignal:      service.StopSignal,
				TTY:             service.Tty,
				OpenStdin:       service.StdinOpen,
				Secrets:         secrets,
				Configs:         configs,
				ReadOnly:        service.ReadOnly,
				Privileges:      &privileges,
				Isolation:       container.Isolation(service.Isolation),
				Init:            service.Init,
				Sysctls:         service.Sysctls,
			},
			LogDriver:     logDriver,
			Resources:     resources,
			RestartPolicy: restartPolicy,
			Placement: &swarm.Placement{
				Constraints: service.Deploy.Placement.Constraints,
				Preferences: getPlacementPreference(service.Deploy.Placement.Preferences),
				MaxReplicas: service.Deploy.Placement.MaxReplicas,
			},
		},
		EndpointSpec:   endpoint,
		Mode:           mode,
		UpdateConfig:   convertUpdateConfig(service.Deploy.UpdateConfig),
		RollbackConfig: convertUpdateConfig(service.Deploy.RollbackConfig),
	}

	// add an image label to serviceSpec
	serviceSpec.Labels[LabelImage] = service.Image

	// ServiceSpec.Networks is deprecated and should not have been used by
	// this package. It is possible to update TaskTemplate.Networks, but it
	// is not possible to update ServiceSpec.Networks. Unfortunately, we
	// can't unconditionally start using TaskTemplate.Networks, because that
	// will break with older daemons that don't support migrating from
	// ServiceSpec.Networks to TaskTemplate.Networks. So which field to use
	// is conditional on daemon version.
	if versions.LessThan(apiVersion, "1.29") {
		serviceSpec.Networks = networks
	} else {
		serviceSpec.TaskTemplate.Networks = networks
	}
	return serviceSpec, nil
}

func getPlacementPreference(preferences []composetypes.PlacementPreferences) []swarm.PlacementPreference {
	result := []swarm.PlacementPreference{}
	for _, preference := range preferences {
		spreadDescriptor := preference.Spread
		result = append(result, swarm.PlacementPreference{
			Spread: &swarm.SpreadOver{
				SpreadDescriptor: spreadDescriptor,
			},
		})
	}
	return result
}

func sortStrings(strs []string) []string {
	sort.Strings(strs)
	return strs
}

func convertServiceNetworks(
	networks map[string]*composetypes.ServiceNetworkConfig,
	networkConfigs networkMap,
	namespace Namespace,
	name string,
) ([]swarm.NetworkAttachmentConfig, error) {
	if len(networks) == 0 {
		networks = map[string]*composetypes.ServiceNetworkConfig{
			defaultNetwork: {},
		}
	}

	nets := []swarm.NetworkAttachmentConfig{}
	for networkName, network := range networks {
		networkConfig, ok := networkConfigs[networkName]
		if !ok && networkName != defaultNetwork {
			return nil, errors.Errorf("undefined network %q", networkName)
		}
		var aliases []string
		if network != nil {
			aliases = network.Aliases
		}
		target := namespace.Scope(networkName)
		if networkConfig.Name != "" {
			target = networkConfig.Name
		}
		netAttachConfig := swarm.NetworkAttachmentConfig{
			Target:  target,
			Aliases: aliases,
		}
		// Only add default aliases to user defined networks. Other networks do
		// not support aliases.
		if container.NetworkMode(target).IsUserDefined() {
			netAttachConfig.Aliases = append(netAttachConfig.Aliases, name)
		}
		nets = append(nets, netAttachConfig)
	}

	sort.Slice(nets, func(i, j int) bool {
		return nets[i].Target < nets[j].Target
	})
	return nets, nil
}

// TODO: fix secrets API so that SecretAPIClient is not required here
func convertServiceSecrets(
	client client.SecretAPIClient,
	namespace Namespace,
	secrets []composetypes.ServiceSecretConfig,
	secretSpecs map[string]composetypes.SecretConfig,
) ([]*swarm.SecretReference, error) {
	refs := []*swarm.SecretReference{}

	lookup := func(key string) (composetypes.FileObjectConfig, error) {
		secretSpec, exists := secretSpecs[key]
		if !exists {
			return composetypes.FileObjectConfig{}, errors.Errorf("undefined secret %q", key)
		}
		return composetypes.FileObjectConfig(secretSpec), nil
	}
	for _, secret := range secrets {
		obj, err := convertFileObject(namespace, composetypes.FileReferenceConfig(secret), lookup)
		if err != nil {
			return nil, err
		}

		file := swarm.SecretReferenceFileTarget(obj.File)
		refs = append(refs, &swarm.SecretReference{
			File:       &file,
			SecretName: obj.Name,
		})
	}

	secrs, err := servicecli.ParseSecrets(client, refs)
	if err != nil {
		return nil, err
	}
	// sort to ensure idempotence (don't restart services just because the entries are in different order)
	sort.SliceStable(secrs, func(i, j int) bool { return secrs[i].SecretName < secrs[j].SecretName })
	return secrs, err
}

// convertServiceConfigObjs takes an API client, a namespace, a ServiceConfig,
// and a set of compose Config specs, and creates the swarm ConfigReferences
// required by the serivce. Unlike convertServiceSecrets, this takes the whole
// ServiceConfig, because some Configs may be needed as a result of other
// fields (like CredentialSpecs).
//
// TODO: fix configs API so that ConfigsAPIClient is not required here
func convertServiceConfigObjs(
	client client.ConfigAPIClient,
	namespace Namespace,
	service composetypes.ServiceConfig,
	configSpecs map[string]composetypes.ConfigObjConfig,
) ([]*swarm.ConfigReference, error) {
	refs := []*swarm.ConfigReference{}

	lookup := func(key string) (composetypes.FileObjectConfig, error) {
		configSpec, exists := configSpecs[key]
		if !exists {
			return composetypes.FileObjectConfig{}, errors.Errorf("undefined config %q", key)
		}
		return composetypes.FileObjectConfig(configSpec), nil
	}
	for _, config := range service.Configs {
		obj, err := convertFileObject(namespace, composetypes.FileReferenceConfig(config), lookup)
		if err != nil {
			return nil, err
		}

		file := swarm.ConfigReferenceFileTarget(obj.File)
		refs = append(refs, &swarm.ConfigReference{
			File:       &file,
			ConfigName: obj.Name,
		})
	}

	// finally, after converting all of the file objects, create any
	// Runtime-type configs that are needed. these are configs that are not
	// mounted into the container, but are used in some other way by the
	// container runtime. Currently, this only means CredentialSpecs, but in
	// the future it may be used for other fields

	// grab the CredentialSpec out of the Service
	credSpec := service.CredentialSpec
	// if the credSpec uses a config, then we should grab the config name, and
	// create a config reference for it. A File or Registry-type CredentialSpec
	// does not need this operation.
	if credSpec.Config != "" {
		// look up the config in the configSpecs.
		obj, err := lookup(credSpec.Config)
		if err != nil {
			return nil, err
		}

		// get the actual correct name.
		name := namespace.Scope(credSpec.Config)
		if obj.Name != "" {
			name = obj.Name
		}

		// now append a Runtime-type config.
		refs = append(refs, &swarm.ConfigReference{
			ConfigName: name,
			Runtime:    &swarm.ConfigReferenceRuntimeTarget{},
		})

	}

	confs, err := servicecli.ParseConfigs(client, refs)
	if err != nil {
		return nil, err
	}
	// sort to ensure idempotence (don't restart services just because the entries are in different order)
	sort.SliceStable(confs, func(i, j int) bool { return confs[i].ConfigName < confs[j].ConfigName })
	return confs, err
}

type swarmReferenceTarget struct {
	Name string
	UID  string
	GID  string
	Mode os.FileMode
}

type swarmReferenceObject struct {
	File swarmReferenceTarget
	ID   string
	Name string
}

func convertFileObject(
	namespace Namespace,
	config composetypes.FileReferenceConfig,
	lookup func(key string) (composetypes.FileObjectConfig, error),
) (swarmReferenceObject, error) {
	obj, err := lookup(config.Source)
	if err != nil {
		return swarmReferenceObject{}, err
	}

	source := namespace.Scope(config.Source)
	if obj.Name != "" {
		source = obj.Name
	}

	target := config.Target
	if target == "" {
		target = config.Source
	}

	uid := config.UID
	gid := config.GID
	if uid == "" {
		uid = "0"
	}
	if gid == "" {
		gid = "0"
	}
	mode := config.Mode
	if mode == nil {
		mode = uint32Ptr(0444)
	}

	return swarmReferenceObject{
		File: swarmReferenceTarget{
			Name: target,
			UID:  uid,
			GID:  gid,
			Mode: os.FileMode(*mode),
		},
		Name: source,
	}, nil
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}

// convertExtraHosts converts <host>:<ip> mappings to SwarmKit notation:
// "IP-address hostname(s)". The original order of mappings is preserved.
func convertExtraHosts(extraHosts composetypes.HostsList) []string {
	hosts := []string{}
	for _, hostIP := range extraHosts {
		if v := strings.SplitN(hostIP, ":", 2); len(v) == 2 {
			// Convert to SwarmKit notation: IP-address hostname(s)
			hosts = append(hosts, fmt.Sprintf("%s %s", v[1], v[0]))
		}
	}
	return hosts
}

func convertHealthcheck(healthcheck *composetypes.HealthCheckConfig) (*container.HealthConfig, error) {
	if healthcheck == nil {
		return nil, nil
	}
	var (
		timeout, interval, startPeriod time.Duration
		retries                        int
	)
	if healthcheck.Disable {
		if len(healthcheck.Test) != 0 {
			return nil, errors.Errorf("test and disable can't be set at the same time")
		}
		return &container.HealthConfig{
			Test: []string{"NONE"},
		}, nil

	}
	if healthcheck.Timeout != nil {
		timeout = time.Duration(*healthcheck.Timeout)
	}
	if healthcheck.Interval != nil {
		interval = time.Duration(*healthcheck.Interval)
	}
	if healthcheck.StartPeriod != nil {
		startPeriod = time.Duration(*healthcheck.StartPeriod)
	}
	if healthcheck.Retries != nil {
		retries = int(*healthcheck.Retries)
	}
	return &container.HealthConfig{
		Test:        healthcheck.Test,
		Timeout:     timeout,
		Interval:    interval,
		Retries:     retries,
		StartPeriod: startPeriod,
	}, nil
}

func convertRestartPolicy(restart string, source *composetypes.RestartPolicy) (*swarm.RestartPolicy, error) {
	// TODO: log if restart is being ignored
	if source == nil {
		policy, err := opts.ParseRestartPolicy(restart)
		if err != nil {
			return nil, err
		}
		switch {
		case policy.IsNone():
			return nil, nil
		case policy.IsAlways(), policy.IsUnlessStopped():
			return &swarm.RestartPolicy{
				Condition: swarm.RestartPolicyConditionAny,
			}, nil
		case policy.IsOnFailure():
			attempts := uint64(policy.MaximumRetryCount)
			return &swarm.RestartPolicy{
				Condition:   swarm.RestartPolicyConditionOnFailure,
				MaxAttempts: &attempts,
			}, nil
		default:
			return nil, errors.Errorf("unknown restart policy: %s", restart)
		}
	}

	return &swarm.RestartPolicy{
		Condition:   swarm.RestartPolicyCondition(source.Condition),
		Delay:       composetypes.ConvertDurationPtr(source.Delay),
		MaxAttempts: source.MaxAttempts,
		Window:      composetypes.ConvertDurationPtr(source.Window),
	}, nil
}

func convertUpdateConfig(source *composetypes.UpdateConfig) *swarm.UpdateConfig {
	if source == nil {
		return nil
	}
	parallel := uint64(1)
	if source.Parallelism != nil {
		parallel = *source.Parallelism
	}
	return &swarm.UpdateConfig{
		Parallelism:     parallel,
		Delay:           time.Duration(source.Delay),
		FailureAction:   source.FailureAction,
		Monitor:         time.Duration(source.Monitor),
		MaxFailureRatio: source.MaxFailureRatio,
		Order:           source.Order,
	}
}

func convertResources(source composetypes.Resources) (*swarm.ResourceRequirements, error) {
	resources := &swarm.ResourceRequirements{}
	var err error
	if source.Limits != nil {
		var cpus int64
		if source.Limits.NanoCPUs != "" {
			cpus, err = opts.ParseCPUs(source.Limits.NanoCPUs)
			if err != nil {
				return nil, err
			}
		}
		resources.Limits = &swarm.Resources{
			NanoCPUs:    cpus,
			MemoryBytes: int64(source.Limits.MemoryBytes),
		}
	}
	if source.Reservations != nil {
		var cpus int64
		if source.Reservations.NanoCPUs != "" {
			cpus, err = opts.ParseCPUs(source.Reservations.NanoCPUs)
			if err != nil {
				return nil, err
			}
		}

		var generic []swarm.GenericResource
		for _, res := range source.Reservations.GenericResources {
			var r swarm.GenericResource

			if res.DiscreteResourceSpec != nil {
				r.DiscreteResourceSpec = &swarm.DiscreteGenericResource{
					Kind:  res.DiscreteResourceSpec.Kind,
					Value: res.DiscreteResourceSpec.Value,
				}
			}

			generic = append(generic, r)
		}

		resources.Reservations = &swarm.Resources{
			NanoCPUs:         cpus,
			MemoryBytes:      int64(source.Reservations.MemoryBytes),
			GenericResources: generic,
		}
	}
	return resources, nil
}

func convertEndpointSpec(endpointMode string, source []composetypes.ServicePortConfig) (*swarm.EndpointSpec, error) {
	portConfigs := []swarm.PortConfig{}
	for _, port := range source {
		portConfig := swarm.PortConfig{
			Protocol:      swarm.PortConfigProtocol(port.Protocol),
			TargetPort:    port.Target,
			PublishedPort: port.Published,
			PublishMode:   swarm.PortConfigPublishMode(port.Mode),
		}
		portConfigs = append(portConfigs, portConfig)
	}

	sort.Slice(portConfigs, func(i, j int) bool {
		return portConfigs[i].PublishedPort < portConfigs[j].PublishedPort
	})

	return &swarm.EndpointSpec{
		Mode:  swarm.ResolutionMode(strings.ToLower(endpointMode)),
		Ports: portConfigs,
	}, nil
}

func convertEnvironment(source map[string]*string) []string {
	var output []string

	for name, value := range source {
		switch value {
		case nil:
			output = append(output, name)
		default:
			output = append(output, fmt.Sprintf("%s=%s", name, *value))
		}
	}

	return output
}

func convertDeployMode(mode string, replicas *uint64) (swarm.ServiceMode, error) {
	serviceMode := swarm.ServiceMode{}

	switch mode {
	case "global":
		if replicas != nil {
			return serviceMode, errors.Errorf("replicas can only be used with replicated mode")
		}
		serviceMode.Global = &swarm.GlobalService{}
	case "replicated", "":
		serviceMode.Replicated = &swarm.ReplicatedService{Replicas: replicas}
	default:
		return serviceMode, errors.Errorf("Unknown mode: %s", mode)
	}
	return serviceMode, nil
}

func convertDNSConfig(DNS []string, DNSSearch []string) (*swarm.DNSConfig, error) {
	if DNS != nil || DNSSearch != nil {
		return &swarm.DNSConfig{
			Nameservers: DNS,
			Search:      DNSSearch,
		}, nil
	}
	return nil, nil
}

func convertCredentialSpec(namespace Namespace, spec composetypes.CredentialSpecConfig, refs []*swarm.ConfigReference) (*swarm.CredentialSpec, error) {
	var o []string

	// Config was added in API v1.40
	if spec.Config != "" {
		o = append(o, `"Config"`)
	}
	if spec.File != "" {
		o = append(o, `"File"`)
	}
	if spec.Registry != "" {
		o = append(o, `"Registry"`)
	}
	l := len(o)
	switch {
	case l == 0:
		return nil, nil
	case l == 2:
		return nil, errors.Errorf("invalid credential spec: cannot specify both %s and %s", o[0], o[1])
	case l > 2:
		return nil, errors.Errorf("invalid credential spec: cannot specify both %s, and %s", strings.Join(o[:l-1], ", "), o[l-1])
	}
	swarmCredSpec := swarm.CredentialSpec(spec)
	// if we're using a swarm Config for the credential spec, over-write it
	// here with the config ID
	if swarmCredSpec.Config != "" {
		for _, config := range refs {
			if swarmCredSpec.Config == config.ConfigName {
				swarmCredSpec.Config = config.ConfigID
				return &swarmCredSpec, nil
			}
		}
		// if none of the configs match, try namespacing
		for _, config := range refs {
			if namespace.Scope(swarmCredSpec.Config) == config.ConfigName {
				swarmCredSpec.Config = config.ConfigID
				return &swarmCredSpec, nil
			}
		}
		return nil, errors.Errorf("invalid credential spec: spec specifies config %v, but no such config can be found", swarmCredSpec.Config)
	}
	return &swarmCredSpec, nil
}
