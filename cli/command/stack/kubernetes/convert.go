package kubernetes

import (
	"github.com/docker/cli/kubernetes/labels"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// toConfigMap converts a Compose Config to a Kube ConfigMap.
func toConfigMap(stackName, name, key string, content []byte) *apiv1.ConfigMap {
	return &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				labels.ForStackName: stackName,
			},
		},
		Data: map[string]string{
			key: string(content),
		},
	}
}

// toSecret converts a Compose Secret to a Kube Secret.
func toSecret(stackName, name, key string, content []byte) *apiv1.Secret {
	return &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				labels.ForStackName: stackName,
			},
		},
		Data: map[string][]byte{
			key: content,
		},
	}
}
