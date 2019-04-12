package kubernetes

import (
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/schema"
	composeTypes "github.com/docker/cli/cli/compose/types"
	composetypes "github.com/docker/cli/cli/compose/types"
	latest "github.com/docker/compose-on-kubernetes/api/compose/v1alpha3"
	"github.com/docker/compose-on-kubernetes/api/compose/v1beta1"
	"github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	"github.com/docker/go-connections/nat"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// kubernatesExtraField is an extra field on ServiceConfigs containing kubernetes-specific extensions to compose format
	kubernatesExtraField = "x-kubernetes"
)

// NewStackConverter returns a converter from types.Config (compose) to the specified
// stack version or error out if the version is not supported or existent.
func NewStackConverter(version string) (StackConverter, error) {
	switch version {
	case "v1beta1":
		return stackV1Beta1Converter{}, nil
	case "v1beta2":
		return stackV1Beta2Converter{}, nil
	case "v1alpha3":
		return stackV1Alpha3Converter{}, nil
	default:
		return nil, errors.Errorf("stack version %s unsupported", version)
	}
}

// StackConverter converts a compose types.Config to a Stack
type StackConverter interface {
	FromCompose(stderr io.Writer, name string, cfg *composetypes.Config) (Stack, error)
}

type stackV1Beta1Converter struct{}

func (s stackV1Beta1Converter) FromCompose(stderr io.Writer, name string, cfg *composetypes.Config) (Stack, error) {
	cfg.Version = v1beta1.MaxComposeVersion
	st, err := fromCompose(stderr, name, cfg, v1beta1Capabilities)
	if err != nil {
		return Stack{}, err
	}
	res, err := yaml.Marshal(cfg)
	if err != nil {
		return Stack{}, err
	}
	// reload the result to check that it produced a valid 3.5 compose file
	resparsedConfig, err := loader.ParseYAML(res)
	if err != nil {
		return Stack{}, err
	}
	if err = schema.Validate(resparsedConfig, v1beta1.MaxComposeVersion); err != nil {
		return Stack{}, errors.Wrapf(err, "the compose yaml file is invalid with v%s", v1beta1.MaxComposeVersion)
	}

	st.ComposeFile = string(res)
	return st, nil
}

type stackV1Beta2Converter struct{}

func (s stackV1Beta2Converter) FromCompose(stderr io.Writer, name string, cfg *composetypes.Config) (Stack, error) {
	return fromCompose(stderr, name, cfg, v1beta2Capabilities)
}

type stackV1Alpha3Converter struct{}

func (s stackV1Alpha3Converter) FromCompose(stderr io.Writer, name string, cfg *composetypes.Config) (Stack, error) {
	return fromCompose(stderr, name, cfg, v1alpha3Capabilities)
}

func fromCompose(stderr io.Writer, name string, cfg *composetypes.Config, capabilities composeCapabilities) (Stack, error) {
	spec, err := fromComposeConfig(stderr, cfg, capabilities)
	if err != nil {
		return Stack{}, err
	}
	return Stack{
		Name: name,
		Spec: spec,
	}, nil
}

func loadStackData(composefile string) (*composetypes.Config, error) {
	parsed, err := loader.ParseYAML([]byte(composefile))
	if err != nil {
		return nil, err
	}
	return loader.Load(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{
				Config: parsed,
			},
		},
	})
}

// Conversions from internal stack to different stack compose component versions.
func stackFromV1beta1(in *v1beta1.Stack) (Stack, error) {
	cfg, err := loadStackData(in.Spec.ComposeFile)
	if err != nil {
		return Stack{}, err
	}
	spec, err := fromComposeConfig(ioutil.Discard, cfg, v1beta1Capabilities)
	if err != nil {
		return Stack{}, err
	}
	return Stack{
		Name:        in.ObjectMeta.Name,
		Namespace:   in.ObjectMeta.Namespace,
		ComposeFile: in.Spec.ComposeFile,
		Spec:        spec,
	}, nil
}

func stackToV1beta1(s Stack) *v1beta1.Stack {
	return &v1beta1.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.Name,
		},
		Spec: v1beta1.StackSpec{
			ComposeFile: s.ComposeFile,
		},
	}
}

func stackFromV1beta2(in *v1beta2.Stack) (Stack, error) {
	var spec *latest.StackSpec
	if in.Spec != nil {
		spec = &latest.StackSpec{}
		if err := latest.Convert_v1beta2_StackSpec_To_v1alpha3_StackSpec(in.Spec, spec, nil); err != nil {
			return Stack{}, err
		}
	}
	return Stack{
		Name:      in.ObjectMeta.Name,
		Namespace: in.ObjectMeta.Namespace,
		Spec:      spec,
	}, nil
}

func stackToV1beta2(s Stack) (*v1beta2.Stack, error) {
	var spec *v1beta2.StackSpec
	if s.Spec != nil {
		spec = &v1beta2.StackSpec{}
		if err := latest.Convert_v1alpha3_StackSpec_To_v1beta2_StackSpec(s.Spec, spec, nil); err != nil {
			return nil, err
		}
	}
	return &v1beta2.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.Name,
		},
		Spec: spec,
	}, nil
}

func stackFromV1alpha3(in *latest.Stack) Stack {
	return Stack{
		Name:      in.ObjectMeta.Name,
		Namespace: in.ObjectMeta.Namespace,
		Spec:      in.Spec,
	}
}

func stackToV1alpha3(s Stack) *latest.Stack {
	return &latest.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.Name,
		},
		Spec: s.Spec,
	}
}

func fromComposeConfig(stderr io.Writer, c *composeTypes.Config, capabilities composeCapabilities) (*latest.StackSpec, error) {
	if c == nil {
		return nil, nil
	}
	warnUnsupportedFeatures(stderr, c)
	serviceConfigs := make([]latest.ServiceConfig, len(c.Services))
	for i, s := range c.Services {
		svc, err := fromComposeServiceConfig(s, capabilities)
		if err != nil {
			return nil, err
		}
		serviceConfigs[i] = svc
	}
	return &latest.StackSpec{
		Services: serviceConfigs,
		Secrets:  fromComposeSecrets(c.Secrets),
		Configs:  fromComposeConfigs(c.Configs),
	}, nil
}

func fromComposeSecrets(s map[string]composeTypes.SecretConfig) map[string]latest.SecretConfig {
	if s == nil {
		return nil
	}
	m := map[string]latest.SecretConfig{}
	for key, value := range s {
		m[key] = latest.SecretConfig{
			Name: value.Name,
			File: value.File,
			External: latest.External{
				Name:     value.External.Name,
				External: value.External.External,
			},
			Labels: value.Labels,
		}
	}
	return m
}

func fromComposeConfigs(s map[string]composeTypes.ConfigObjConfig) map[string]latest.ConfigObjConfig {
	if s == nil {
		return nil
	}
	m := map[string]latest.ConfigObjConfig{}
	for key, value := range s {
		m[key] = latest.ConfigObjConfig{
			Name: value.Name,
			File: value.File,
			External: latest.External{
				Name:     value.External.Name,
				External: value.External.External,
			},
			Labels: value.Labels,
		}
	}
	return m
}

func fromComposeServiceConfig(s composeTypes.ServiceConfig, capabilities composeCapabilities) (latest.ServiceConfig, error) {
	var (
		userID *int64
		err    error
	)
	if s.User != "" {
		numerical, err := strconv.Atoi(s.User)
		if err == nil {
			unixUserID := int64(numerical)
			userID = &unixUserID
		}
	}
	kubeExtra, err := resolveServiceExtra(s)
	if err != nil {
		return latest.ServiceConfig{}, err
	}
	if kubeExtra.PullSecret != "" && !capabilities.hasPullSecrets {
		return latest.ServiceConfig{}, errors.Errorf(`stack API version %s does not support pull secrets (field "x-kubernetes.pull_secret"), please use version v1alpha3 or higher`, capabilities.apiVersion)
	}
	if kubeExtra.PullPolicy != "" && !capabilities.hasPullPolicies {
		return latest.ServiceConfig{}, errors.Errorf(`stack API version %s does not support pull policies (field "x-kubernetes.pull_policy"), please use version v1alpha3 or higher`, capabilities.apiVersion)
	}

	internalPorts, err := setupIntraStackNetworking(s, kubeExtra, capabilities)
	if err != nil {
		return latest.ServiceConfig{}, err
	}

	return latest.ServiceConfig{
		Name:    s.Name,
		CapAdd:  s.CapAdd,
		CapDrop: s.CapDrop,
		Command: s.Command,
		Configs: fromComposeServiceConfigs(s.Configs),
		Deploy: latest.DeployConfig{
			Mode:          s.Deploy.Mode,
			Replicas:      s.Deploy.Replicas,
			Labels:        s.Deploy.Labels,
			UpdateConfig:  fromComposeUpdateConfig(s.Deploy.UpdateConfig),
			Resources:     fromComposeResources(s.Deploy.Resources),
			RestartPolicy: fromComposeRestartPolicy(s.Deploy.RestartPolicy),
			Placement:     fromComposePlacement(s.Deploy.Placement),
		},
		Entrypoint:          s.Entrypoint,
		Environment:         s.Environment,
		ExtraHosts:          s.ExtraHosts,
		Hostname:            s.Hostname,
		HealthCheck:         fromComposeHealthcheck(s.HealthCheck),
		Image:               s.Image,
		Ipc:                 s.Ipc,
		Labels:              s.Labels,
		Pid:                 s.Pid,
		Ports:               fromComposePorts(s.Ports),
		Privileged:          s.Privileged,
		ReadOnly:            s.ReadOnly,
		Secrets:             fromComposeServiceSecrets(s.Secrets),
		StdinOpen:           s.StdinOpen,
		StopGracePeriod:     composetypes.ConvertDurationPtr(s.StopGracePeriod),
		Tmpfs:               s.Tmpfs,
		Tty:                 s.Tty,
		User:                userID,
		Volumes:             fromComposeServiceVolumeConfig(s.Volumes),
		WorkingDir:          s.WorkingDir,
		PullSecret:          kubeExtra.PullSecret,
		PullPolicy:          kubeExtra.PullPolicy,
		InternalServiceType: kubeExtra.InternalServiceType,
		InternalPorts:       internalPorts,
	}, nil
}

func setupIntraStackNetworking(s composeTypes.ServiceConfig, kubeExtra kubernetesExtra, capabilities composeCapabilities) ([]latest.InternalPort, error) {
	if kubeExtra.InternalServiceType != latest.InternalServiceTypeAuto && !capabilities.hasIntraStackLoadBalancing {
		return nil,
			errors.Errorf(`stack API version %s does not support intra-stack load balancing (field "x-kubernetes.internal_service_type"), please use version v1alpha3 or higher`,
				capabilities.apiVersion)
	}
	if !capabilities.hasIntraStackLoadBalancing {
		return nil, nil
	}
	if err := validateInternalServiceType(kubeExtra.InternalServiceType); err != nil {
		return nil, err
	}
	internalPorts, err := toInternalPorts(s.Expose)
	if err != nil {
		return nil, err
	}
	return internalPorts, nil
}

func validateInternalServiceType(internalServiceType latest.InternalServiceType) error {
	switch internalServiceType {
	case latest.InternalServiceTypeAuto, latest.InternalServiceTypeClusterIP, latest.InternalServiceTypeHeadless:
	default:
		return errors.Errorf(`invalid value %q for field "x-kubernetes.internal_service_type", valid values are %q or %q`, internalServiceType,
			latest.InternalServiceTypeClusterIP,
			latest.InternalServiceTypeHeadless)
	}
	return nil
}

func toInternalPorts(expose []string) ([]latest.InternalPort, error) {
	var internalPorts []latest.InternalPort
	for _, sourcePort := range expose {
		proto, port := nat.SplitProtoPort(sourcePort)
		start, end, err := nat.ParsePortRange(port)
		if err != nil {
			return nil, errors.Errorf("invalid format for expose: %q, error: %s", sourcePort, err)
		}
		for i := start; i <= end; i++ {
			k8sProto := v1.Protocol(strings.ToUpper(proto))
			switch k8sProto {
			case v1.ProtocolSCTP, v1.ProtocolTCP, v1.ProtocolUDP:
			default:
				return nil, errors.Errorf("invalid protocol for expose: %q, supported values are %q, %q and %q", sourcePort, v1.ProtocolSCTP, v1.ProtocolTCP, v1.ProtocolUDP)
			}
			internalPorts = append(internalPorts, latest.InternalPort{
				Port:     int32(i),
				Protocol: k8sProto,
			})
		}
	}
	return internalPorts, nil
}

func resolveServiceExtra(s composeTypes.ServiceConfig) (kubernetesExtra, error) {
	if iface, ok := s.Extras[kubernatesExtraField]; ok {
		var result kubernetesExtra
		if err := mapstructure.Decode(iface, &result); err != nil {
			return kubernetesExtra{}, err
		}
		return result, nil
	}
	return kubernetesExtra{}, nil
}

func fromComposePorts(ports []composeTypes.ServicePortConfig) []latest.ServicePortConfig {
	if ports == nil {
		return nil
	}
	p := make([]latest.ServicePortConfig, len(ports))
	for i, port := range ports {
		p[i] = latest.ServicePortConfig{
			Mode:      port.Mode,
			Target:    port.Target,
			Published: port.Published,
			Protocol:  port.Protocol,
		}
	}
	return p
}

func fromComposeServiceSecrets(secrets []composeTypes.ServiceSecretConfig) []latest.ServiceSecretConfig {
	if secrets == nil {
		return nil
	}
	c := make([]latest.ServiceSecretConfig, len(secrets))
	for i, secret := range secrets {
		c[i] = latest.ServiceSecretConfig{
			Source: secret.Source,
			Target: secret.Target,
			UID:    secret.UID,
			Mode:   secret.Mode,
		}
	}
	return c
}

func fromComposeServiceConfigs(configs []composeTypes.ServiceConfigObjConfig) []latest.ServiceConfigObjConfig {
	if configs == nil {
		return nil
	}
	c := make([]latest.ServiceConfigObjConfig, len(configs))
	for i, config := range configs {
		c[i] = latest.ServiceConfigObjConfig{
			Source: config.Source,
			Target: config.Target,
			UID:    config.UID,
			Mode:   config.Mode,
		}
	}
	return c
}

func fromComposeHealthcheck(h *composeTypes.HealthCheckConfig) *latest.HealthCheckConfig {
	if h == nil {
		return nil
	}
	return &latest.HealthCheckConfig{
		Test:     h.Test,
		Timeout:  composetypes.ConvertDurationPtr(h.Timeout),
		Interval: composetypes.ConvertDurationPtr(h.Interval),
		Retries:  h.Retries,
	}
}

func fromComposePlacement(p composeTypes.Placement) latest.Placement {
	return latest.Placement{
		Constraints: fromComposeConstraints(p.Constraints),
	}
}

var constraintEquals = regexp.MustCompile(`([\w\.]*)\W*(==|!=)\W*([\w\.]*)`)

const (
	swarmOs          = "node.platform.os"
	swarmArch        = "node.platform.arch"
	swarmHostname    = "node.hostname"
	swarmLabelPrefix = "node.labels."
)

func fromComposeConstraints(s []string) *latest.Constraints {
	if len(s) == 0 {
		return nil
	}
	constraints := &latest.Constraints{}
	for _, constraint := range s {
		matches := constraintEquals.FindStringSubmatch(constraint)
		if len(matches) == 4 {
			key := matches[1]
			operator := matches[2]
			value := matches[3]
			constraint := &latest.Constraint{
				Operator: operator,
				Value:    value,
			}
			switch {
			case key == swarmOs:
				constraints.OperatingSystem = constraint
			case key == swarmArch:
				constraints.Architecture = constraint
			case key == swarmHostname:
				constraints.Hostname = constraint
			case strings.HasPrefix(key, swarmLabelPrefix):
				if constraints.MatchLabels == nil {
					constraints.MatchLabels = map[string]latest.Constraint{}
				}
				constraints.MatchLabels[strings.TrimPrefix(key, swarmLabelPrefix)] = *constraint
			}
		}
	}
	return constraints
}

func fromComposeResources(r composeTypes.Resources) latest.Resources {
	return latest.Resources{
		Limits:       fromComposeResourcesResource(r.Limits),
		Reservations: fromComposeResourcesResource(r.Reservations),
	}
}

func fromComposeResourcesResource(r *composeTypes.Resource) *latest.Resource {
	if r == nil {
		return nil
	}
	return &latest.Resource{
		MemoryBytes: int64(r.MemoryBytes),
		NanoCPUs:    r.NanoCPUs,
	}
}

func fromComposeUpdateConfig(u *composeTypes.UpdateConfig) *latest.UpdateConfig {
	if u == nil {
		return nil
	}
	return &latest.UpdateConfig{
		Parallelism: u.Parallelism,
	}
}

func fromComposeRestartPolicy(r *composeTypes.RestartPolicy) *latest.RestartPolicy {
	if r == nil {
		return nil
	}
	return &latest.RestartPolicy{
		Condition: r.Condition,
	}
}

func fromComposeServiceVolumeConfig(vs []composeTypes.ServiceVolumeConfig) []latest.ServiceVolumeConfig {
	if vs == nil {
		return nil
	}
	volumes := []latest.ServiceVolumeConfig{}
	for _, v := range vs {
		volumes = append(volumes, latest.ServiceVolumeConfig{
			Type:     v.Type,
			Source:   v.Source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		})
	}
	return volumes
}

var (
	v1beta1Capabilities = composeCapabilities{
		apiVersion: "v1beta1",
	}
	v1beta2Capabilities = composeCapabilities{
		apiVersion: "v1beta2",
	}
	v1alpha3Capabilities = composeCapabilities{
		apiVersion:                 "v1alpha3",
		hasPullSecrets:             true,
		hasPullPolicies:            true,
		hasIntraStackLoadBalancing: true,
	}
)

type composeCapabilities struct {
	apiVersion                 string
	hasPullSecrets             bool
	hasPullPolicies            bool
	hasIntraStackLoadBalancing bool
}

type kubernetesExtra struct {
	PullSecret          string                     `mapstructure:"pull_secret"`
	PullPolicy          string                     `mapstructure:"pull_policy"`
	InternalServiceType latest.InternalServiceType `mapstructure:"internal_service_type"`
}
