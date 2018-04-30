package kubernetes

import (
	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	apiv1beta2 "github.com/docker/cli/kubernetes/compose/v1beta2"
	"github.com/pkg/errors"
	apimachinerymetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

// StackVersion represents the detected Compose Component on Kubernetes side.
type StackVersion string

const (
	// StackAPIV1Beta1 is returned if it's the most recent version available.
	StackAPIV1Beta1 = StackVersion("v1beta1")
	// StackAPIV1Beta2 is returned if it's the most recent version available.
	StackAPIV1Beta2 = StackVersion("v1beta2")
)

// GetStackAPIVersion returns the most recent stack API installed.
func GetStackAPIVersion(clientSet *kubernetes.Clientset) (StackVersion, error) {
	groups, err := clientSet.Discovery().ServerGroups()
	if err != nil {
		return "", err
	}

	return getAPIVersion(groups)
}

func getAPIVersion(groups *metav1.APIGroupList) (StackVersion, error) {
	switch {
	case findVersion(apiv1beta2.SchemeGroupVersion, groups.Groups):
		return StackAPIV1Beta2, nil
	case findVersion(apiv1beta1.SchemeGroupVersion, groups.Groups):
		return StackAPIV1Beta1, nil
	default:
		return "", errors.Errorf("failed to find a Stack API version")
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
