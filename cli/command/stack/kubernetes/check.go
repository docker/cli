package kubernetes

import (
	"fmt"

	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// APIPresent checks that an API is installed.
func APIPresent(config *rest.Config) error {
	log.Debugf("check API present at %s", config.Host)
	clients, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	groups, err := clients.Discovery().ServerGroups()
	if err != nil {
		return err
	}

	for _, group := range groups.Groups {
		if group.Name == apiv1beta1.SchemeGroupVersion.Group {
			return nil
		}
	}

	return fmt.Errorf("could not find %s api. Install it on your cluster first", apiv1beta1.SchemeGroupVersion.Group)
}
