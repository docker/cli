package kubernetes

import (
	"fmt"
	"sort"

	composetypes "github.com/docker/cli/cli/compose/types"
	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	"github.com/docker/cli/kubernetes/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// IsColliding verify that services defined in the stack collides with already deployed services
func IsColliding(services corev1.ServiceInterface, stack *apiv1beta1.Stack, cfg *composetypes.Config) error {
	stackObjects := getServices(cfg)

	for _, srv := range stackObjects {
		if err := verify(services, stack.Name, srv); err != nil {
			return err
		}
	}

	return nil
}

// verify checks wether the service is already present in kubernetes.
// If we find the service by name but it doesn't have our label or it has a different value
// than the stack name for the label, we fail (i.e. it will collide)
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

func getServices(cfg *composetypes.Config) []string {
	services := make([]string, len(cfg.Services))
	for i := range cfg.Services {
		services[i] = cfg.Services[i].Name
	}
	sort.Strings(services)
	return services
}
