package kubernetes

import (
	"io/ioutil"
	"path/filepath"
	"sort"

	"github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	"github.com/docker/compose-on-kubernetes/api/labels"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Stack is the main type used by stack commands so they remain independent from kubernetes compose component version.
type Stack struct {
	Name        string
	Namespace   string
	ComposeFile string
	Spec        *v1beta2.StackSpec
}

// getServices returns all the stack service names, sorted lexicographically
func (s *Stack) getServices() []string {
	services := make([]string, len(s.Spec.Services))
	for i, service := range s.Spec.Services {
		services[i] = service.Name
	}
	sort.Strings(services)
	return services
}

// createFileBasedConfigMaps creates a Kubernetes ConfigMap for each Compose global file-based config.
func (s *Stack) createFileBasedConfigMaps(configMaps corev1.ConfigMapInterface) error {
	for name, config := range s.Spec.Configs {
		if config.File == "" {
			continue
		}

		fileName := filepath.Base(config.File)
		content, err := ioutil.ReadFile(config.File)
		if err != nil {
			return err
		}

		if _, err := configMaps.Create(toConfigMap(s.Name, name, fileName, content)); err != nil {
			return err
		}
	}

	return nil
}

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

// createFileBasedSecrets creates a Kubernetes Secret for each Compose global file-based secret.
func (s *Stack) createFileBasedSecrets(secrets corev1.SecretInterface) error {
	for name, secret := range s.Spec.Secrets {
		if secret.File == "" {
			continue
		}

		fileName := filepath.Base(secret.File)
		content, err := ioutil.ReadFile(secret.File)
		if err != nil {
			return err
		}

		if _, err := secrets.Create(toSecret(s.Name, name, fileName, content)); err != nil {
			return err
		}
	}

	return nil
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
