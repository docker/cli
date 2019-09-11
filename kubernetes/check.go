package kubernetes

import (
	apiv1alpha3 "github.com/docker/compose-on-kubernetes/api/compose/v1alpha3"
	apiv1beta1 "github.com/docker/compose-on-kubernetes/api/compose/v1beta1"
	apiv1beta2 "github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	"github.com/pkg/errors"
	apimachinerymetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// StackVersion represents the detected Compose Component on Kubernetes side.
type StackVersion string

const (
	// StackAPIV1Beta1 is returned if it's the most recent version available.
	StackAPIV1Beta1 = StackVersion("v1beta1")
	// StackAPIV1Beta2 is returned if it's the most recent version available.
	StackAPIV1Beta2 = StackVersion("v1beta2")
	// StackAPIV1Alpha3 is returned if it's the most recent version available, and experimental flag is on.
	StackAPIV1Alpha3 = StackVersion("v1alpha3")
)

// GetStackAPIVersion returns the most appropriate stack API version installed.
func GetStackAPIVersion(serverGroups discovery.ServerGroupsInterface, experimental bool) (StackVersion, error) {
	groups, err := serverGroups.ServerGroups()
	if err != nil {
		return "", err
	}

	return getAPIVersion(groups, experimental)
}

func getAPIVersion(groups *metav1.APIGroupList, experimental bool) (StackVersion, error) {
	switch {
	case experimental && findVersion(apiv1alpha3.SchemeGroupVersion, groups.Groups):
		return StackAPIV1Alpha3, nil
	case findVersion(apiv1beta2.SchemeGroupVersion, groups.Groups):
		return StackAPIV1Beta2, nil
	case findVersion(apiv1beta1.SchemeGroupVersion, groups.Groups):
		return StackAPIV1Beta1, nil
	default:
		return "", errors.New("failed to find a Stack API version")
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
