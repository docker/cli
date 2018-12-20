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
	"github.com/docker/compose-on-kubernetes/api/compose/v1beta1"
	"github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewStackConverter returns a converter from types.Config (compose) to the specified
// stack version or error out if the version is not supported or existent.
func NewStackConverter(version string) (StackConverter, error) {
	switch version {
	case "v1beta1":
		return stackV1Beta1Converter{}, nil
	case "v1beta2":
		return stackV1Beta2Converter{}, nil
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
	st, err := fromCompose(stderr, name, cfg)
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
	return fromCompose(stderr, name, cfg)
}

func fromCompose(stderr io.Writer, name string, cfg *composetypes.Config) (Stack, error) {
	return Stack{
		Name: name,
		Spec: fromComposeConfig(stderr, cfg),
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
	return Stack{
		Name:        in.ObjectMeta.Name,
		Namespace:   in.ObjectMeta.Namespace,
		ComposeFile: in.Spec.ComposeFile,
		Spec:        fromComposeConfig(ioutil.Discard, cfg),
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

func stackFromV1beta2(in *v1beta2.Stack) Stack {
	return Stack{
		Name:      in.ObjectMeta.Name,
		Namespace: in.ObjectMeta.Namespace,
		Spec:      in.Spec,
	}
}

func stackToV1beta2(s Stack) *v1beta2.Stack {
	return &v1beta2.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.Name,
		},
		Spec: s.Spec,
	}
}

func fromComposeConfig(stderr io.Writer, c *composeTypes.Config) *v1beta2.StackSpec {
	if c == nil {
		return nil
	}
	warnUnsupportedFeatures(stderr, c)
	serviceConfigs := make([]v1beta2.ServiceConfig, len(c.Services))
	for i, s := range c.Services {
		serviceConfigs[i] = fromComposeServiceConfig(s)
	}
	return &v1beta2.StackSpec{
		Services: serviceConfigs,
		Secrets:  fromComposeSecrets(c.Secrets),
		Configs:  fromComposeConfigs(c.Configs),
	}
}

func fromComposeSecrets(s map[string]composeTypes.SecretConfig) map[string]v1beta2.SecretConfig {
	if s == nil {
		return nil
	}
	m := map[string]v1beta2.SecretConfig{}
	for key, value := range s {
		m[key] = v1beta2.SecretConfig{
			Name: value.Name,
			File: value.File,
			External: v1beta2.External{
				Name:     value.External.Name,
				External: value.External.External,
			},
			Labels: value.Labels,
		}
	}
	return m
}

func fromComposeConfigs(s map[string]composeTypes.ConfigObjConfig) map[string]v1beta2.ConfigObjConfig {
	if s == nil {
		return nil
	}
	m := map[string]v1beta2.ConfigObjConfig{}
	for key, value := range s {
		m[key] = v1beta2.ConfigObjConfig{
			Name: value.Name,
			File: value.File,
			External: v1beta2.External{
				Name:     value.External.Name,
				External: value.External.External,
			},
			Labels: value.Labels,
		}
	}
	return m
}

func fromComposeServiceConfig(s composeTypes.ServiceConfig) v1beta2.ServiceConfig {
	var userID *int64
	if s.User != "" {
		numerical, err := strconv.Atoi(s.User)
		if err == nil {
			unixUserID := int64(numerical)
			userID = &unixUserID
		}
	}
	return v1beta2.ServiceConfig{
		Name:    s.Name,
		CapAdd:  s.CapAdd,
		CapDrop: s.CapDrop,
		Command: s.Command,
		Configs: fromComposeServiceConfigs(s.Configs),
		Deploy: v1beta2.DeployConfig{
			Mode:          s.Deploy.Mode,
			Replicas:      s.Deploy.Replicas,
			Labels:        s.Deploy.Labels,
			UpdateConfig:  fromComposeUpdateConfig(s.Deploy.UpdateConfig),
			Resources:     fromComposeResources(s.Deploy.Resources),
			RestartPolicy: fromComposeRestartPolicy(s.Deploy.RestartPolicy),
			Placement:     fromComposePlacement(s.Deploy.Placement),
		},
		Entrypoint:      s.Entrypoint,
		Environment:     s.Environment,
		ExtraHosts:      s.ExtraHosts,
		Hostname:        s.Hostname,
		HealthCheck:     fromComposeHealthcheck(s.HealthCheck),
		Image:           s.Image,
		Ipc:             s.Ipc,
		Labels:          s.Labels,
		Pid:             s.Pid,
		Ports:           fromComposePorts(s.Ports),
		Privileged:      s.Privileged,
		ReadOnly:        s.ReadOnly,
		Secrets:         fromComposeServiceSecrets(s.Secrets),
		StdinOpen:       s.StdinOpen,
		StopGracePeriod: composetypes.ConvertDurationPtr(s.StopGracePeriod),
		Tmpfs:           s.Tmpfs,
		Tty:             s.Tty,
		User:            userID,
		Volumes:         fromComposeServiceVolumeConfig(s.Volumes),
		WorkingDir:      s.WorkingDir,
	}
}

func fromComposePorts(ports []composeTypes.ServicePortConfig) []v1beta2.ServicePortConfig {
	if ports == nil {
		return nil
	}
	p := make([]v1beta2.ServicePortConfig, len(ports))
	for i, port := range ports {
		p[i] = v1beta2.ServicePortConfig{
			Mode:      port.Mode,
			Target:    port.Target,
			Published: port.Published,
			Protocol:  port.Protocol,
		}
	}
	return p
}

func fromComposeServiceSecrets(secrets []composeTypes.ServiceSecretConfig) []v1beta2.ServiceSecretConfig {
	if secrets == nil {
		return nil
	}
	c := make([]v1beta2.ServiceSecretConfig, len(secrets))
	for i, secret := range secrets {
		c[i] = v1beta2.ServiceSecretConfig{
			Source: secret.Source,
			Target: secret.Target,
			UID:    secret.UID,
			Mode:   secret.Mode,
		}
	}
	return c
}

func fromComposeServiceConfigs(configs []composeTypes.ServiceConfigObjConfig) []v1beta2.ServiceConfigObjConfig {
	if configs == nil {
		return nil
	}
	c := make([]v1beta2.ServiceConfigObjConfig, len(configs))
	for i, config := range configs {
		c[i] = v1beta2.ServiceConfigObjConfig{
			Source: config.Source,
			Target: config.Target,
			UID:    config.UID,
			Mode:   config.Mode,
		}
	}
	return c
}

func fromComposeHealthcheck(h *composeTypes.HealthCheckConfig) *v1beta2.HealthCheckConfig {
	if h == nil {
		return nil
	}
	return &v1beta2.HealthCheckConfig{
		Test:     h.Test,
		Timeout:  composetypes.ConvertDurationPtr(h.Timeout),
		Interval: composetypes.ConvertDurationPtr(h.Interval),
		Retries:  h.Retries,
	}
}

func fromComposePlacement(p composeTypes.Placement) v1beta2.Placement {
	return v1beta2.Placement{
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

func fromComposeConstraints(s []string) *v1beta2.Constraints {
	if len(s) == 0 {
		return nil
	}
	constraints := &v1beta2.Constraints{}
	for _, constraint := range s {
		matches := constraintEquals.FindStringSubmatch(constraint)
		if len(matches) == 4 {
			key := matches[1]
			operator := matches[2]
			value := matches[3]
			constraint := &v1beta2.Constraint{
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
					constraints.MatchLabels = map[string]v1beta2.Constraint{}
				}
				constraints.MatchLabels[strings.TrimPrefix(key, swarmLabelPrefix)] = *constraint
			}
		}
	}
	return constraints
}

func fromComposeResources(r composeTypes.Resources) v1beta2.Resources {
	return v1beta2.Resources{
		Limits:       fromComposeResourcesResource(r.Limits),
		Reservations: fromComposeResourcesResource(r.Reservations),
	}
}

func fromComposeResourcesResource(r *composeTypes.Resource) *v1beta2.Resource {
	if r == nil {
		return nil
	}
	return &v1beta2.Resource{
		MemoryBytes: int64(r.MemoryBytes),
		NanoCPUs:    r.NanoCPUs,
	}
}

func fromComposeUpdateConfig(u *composeTypes.UpdateConfig) *v1beta2.UpdateConfig {
	if u == nil {
		return nil
	}
	return &v1beta2.UpdateConfig{
		Parallelism: u.Parallelism,
	}
}

func fromComposeRestartPolicy(r *composeTypes.RestartPolicy) *v1beta2.RestartPolicy {
	if r == nil {
		return nil
	}
	return &v1beta2.RestartPolicy{
		Condition: r.Condition,
	}
}

func fromComposeServiceVolumeConfig(vs []composeTypes.ServiceVolumeConfig) []v1beta2.ServiceVolumeConfig {
	if vs == nil {
		return nil
	}
	volumes := []v1beta2.ServiceVolumeConfig{}
	for _, v := range vs {
		volumes = append(volumes, v1beta2.ServiceVolumeConfig{
			Type:     v.Type,
			Source:   v.Source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		})
	}
	return volumes
}
