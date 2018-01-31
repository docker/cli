package kubernetes

import (
	"fmt"

	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	apiv1beta2 "github.com/docker/cli/kubernetes/compose/v1beta2"
	log "github.com/sirupsen/logrus"
	apimachinerymetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

type KubernetesStackVersion string

const (
	// KubernetesStackNotFound is returned when no stack api at all is detected on kubernetes.
	KubernetesStackNotFound = "notFound"
	// KubernetesStackAPIV1Beta1 is returned if it's the most recent version available.
	KubernetesStackAPIV1Beta1 = "v1beta1"
	// KubernetesStackAPIV1Beta2 is returned if it's the most recent version available.
	KubernetesStackAPIV1Beta2 = "v1beta2"
)

// GetAPIVersion returns the most recent stack API installed.
func (c *KubeCli) GetAPIVersion() (KubernetesStackVersion, error) {
	log.Debugf("retrieve most recent stack API present at %s", c.KubeConfig.Host)
	clients, err := kubernetes.NewForConfig(c.KubeConfig)
	if err != nil {
		return KubernetesStackNotFound, err
	}

	groups, err := clients.Discovery().ServerGroups()
	if err != nil {
		return KubernetesStackNotFound, err
	}

	switch {
	case findVersion(apiv1beta2.SchemeGroupVersion, groups.Groups):
		return KubernetesStackAPIV1Beta2, nil
	case findVersion(apiv1beta1.SchemeGroupVersion, groups.Groups):
		return KubernetesStackAPIV1Beta1, nil
	default:
		return KubernetesStackNotFound, fmt.Errorf("could not find %s api. Install it on your cluster first", apiv1beta1.SchemeGroupVersion.Group)
	}
}

func findVersion(stackApi schema.GroupVersion, groups []apimachinerymetav1.APIGroup) bool {
	for _, group := range groups {
		if group.Name == stackApi.Group {
			for _, version := range group.Versions {
				if version.Version == stackApi.Version {
					return true
				}
			}
		}
	}
	return false
}
