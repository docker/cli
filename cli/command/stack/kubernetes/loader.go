package kubernetes

import (
	composetypes "github.com/docker/cli/cli/compose/types"
	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LoadStack loads a stack from a Compose config, with a given name.
func LoadStack(name string, cfg composetypes.Config) (*apiv1beta1.Stack, error) {
	res, err := yaml.Marshal(cfg)
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
