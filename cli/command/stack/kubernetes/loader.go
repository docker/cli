package kubernetes

import (
	composetypes "github.com/docker/cli/cli/compose/types"
	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type versionedConfig struct {
	composetypes.Config
	version string
}

func (c versionedConfig) MarshalYAML() (interface{}, error) {
	services := map[string]composetypes.ServiceConfig{}
	for _, service := range c.Services {
		services[service.Name] = service
	}
	return map[string]interface{}{
		"services": services,
		"networks": c.Networks,
		"volumes":  c.Volumes,
		"secrets":  c.Secrets,
		"configs":  c.Configs,
		"version":  c.version,
	}, nil
}

// LoadStack loads a stack from a Compose config, with a given name.
func LoadStack(name, version string, cfg composetypes.Config) (*apiv1beta1.Stack, error) {
	res, err := yaml.Marshal(versionedConfig{
		version: version,
		Config:  cfg,
	})
	if err != nil {
		return nil, err
	}
	return &apiv1beta1.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiv1beta1.StackSpec{
			ComposeFile: string(res),
		},
	}, nil
}
