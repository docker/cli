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

// StackVersion represents the detected Compose Component on Kubernetes side.
type StackVersion string

const (
	// StackNotFound is returned when no stack api at all is detected on kubernetes.
	StackNotFound = "notFound"
	// StackAPIV1Beta1 is returned if it's the most recent version available.
	StackAPIV1Beta1 = "v1beta1"
	// StackAPIV1Beta2 is returned if it's the most recent version available.
	StackAPIV1Beta2 = "v1beta2"
)

// GetAPIVersion returns the most recent stack API installed.
func (c *KubeCli) GetAPIVersion() (StackVersion, error) {
	log.Debugf("retrieve most recent stack API present at %s", c.KubeConfig.Host)
	clients, err := kubernetes.NewForConfig(c.KubeConfig)
	if err != nil {
		return StackNotFound, err
	}

	groups, err := clients.Discovery().ServerGroups()
	if err != nil {
		return StackNotFound, err
	}

	switch {
	case findVersion(apiv1beta2.SchemeGroupVersion, groups.Groups):
		return StackAPIV1Beta2, nil
	case findVersion(apiv1beta1.SchemeGroupVersion, groups.Groups):
		return StackAPIV1Beta1, nil
	default:
		return StackNotFound, fmt.Errorf("could not find %s api. Install it on your cluster first", apiv1beta1.SchemeGroupVersion.Group)
	}
}

func findVersion(stackAPI schema.GroupVersion, groups []apimachinerymetav1.APIGroup) bool {
	for _, group := range groups {
		if group.Name == stackAPI.Group {
			for _, version := range group.Versions {
				if version.Version == stackAPI.Version {
					return true
				}
			}
		}
	}
	return false
}
