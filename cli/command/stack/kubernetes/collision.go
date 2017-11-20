package kubernetes

import (
	"fmt"
	"sort"

	apiv1beta1 "github.com/docker/cli/cli/command/stack/kubernetes/api/compose/v1beta1"
	"github.com/docker/cli/cli/command/stack/kubernetes/api/labels"
	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// IsColliding verify that services defined in the stack collides with already deployed services
func IsColliding(services corev1.ServiceInterface, stack *apiv1beta1.Stack) error {
	stackObjects, err := getServices(stack.Spec.ComposeFile)
	if err != nil {
		return err
	}

	for _, srv := range stackObjects {
		if err := verify(services, stack.Name, srv); err != nil {
			return err
		}
	}

	return nil
}

func verify(services corev1.ServiceInterface, stackName string, service string) error {
	svc, err := services.Get(service, metav1.GetOptions{})
	if err == nil {
		if key, ok := svc.ObjectMeta.Labels[labels.ForStackName]; ok {
			if key != stackName {
				return fmt.Errorf("service %s already present in stack named %s", service, key)
			}

			return nil
		}

		return fmt.Errorf("service %s already present in the cluster", service)
	}

	return nil
}

func getServices(composeFile string) ([]string, error) {
	parsed, err := loader.ParseYAML([]byte(composeFile))
	if err != nil {
		return nil, err
	}

	config, err := loader.Load(types.ConfigDetails{
		WorkingDir: ".",
		ConfigFiles: []types.ConfigFile{
			{
				Config: parsed,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	services := make([]string, len(config.Services))
	for i := range config.Services {
		services[i] = config.Services[i].Name
	}
	sort.Strings(services)
	return services, nil
}
