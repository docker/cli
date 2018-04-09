package kubernetes

import (
	"github.com/docker/cli/kubernetes"
	"github.com/pkg/errors"
	kubeclient "k8s.io/client-go/kubernetes"
	appsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	typesappsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
)

// Factory is the kubernetes client factory
type Factory struct {
	namespace     string
	config        *restclient.Config
	coreClientSet *corev1.CoreV1Client
	appsClientSet *appsv1beta2.AppsV1beta2Client
	clientSet     *kubeclient.Clientset
}

// NewFactory creates a kubernetes client factory
func NewFactory(namespace string, config *restclient.Config, clientSet *kubeclient.Clientset) (*Factory, error) {
	coreClientSet, err := corev1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	appsClientSet, err := appsv1beta2.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Factory{
		namespace:     namespace,
		config:        config,
		coreClientSet: coreClientSet,
		appsClientSet: appsClientSet,
		clientSet:     clientSet,
	}, nil
}

// ConfigMaps returns a client for kubernetes's config maps
func (s *Factory) ConfigMaps() corev1.ConfigMapInterface {
	return s.coreClientSet.ConfigMaps(s.namespace)
}

// Secrets returns a client for kubernetes's secrets
func (s *Factory) Secrets() corev1.SecretInterface {
	return s.coreClientSet.Secrets(s.namespace)
}

// Services returns a client for kubernetes's secrets
func (s *Factory) Services() corev1.ServiceInterface {
	return s.coreClientSet.Services(s.namespace)
}

// Pods returns a client for kubernetes's pods
func (s *Factory) Pods() corev1.PodInterface {
	return s.coreClientSet.Pods(s.namespace)
}

// Nodes returns a client for kubernetes's nodes
func (s *Factory) Nodes() corev1.NodeInterface {
	return s.coreClientSet.Nodes()
}

// ReplicationControllers returns a client for kubernetes replication controllers
func (s *Factory) ReplicationControllers() corev1.ReplicationControllerInterface {
	return s.coreClientSet.ReplicationControllers(s.namespace)
}

// ReplicaSets return a client for kubernetes replace sets
func (s *Factory) ReplicaSets() typesappsv1beta2.ReplicaSetInterface {
	return s.appsClientSet.ReplicaSets(s.namespace)
}

func (c *Factory) Stacks() (stackClient, error) {
	version, err := kubernetes.GetStackAPIVersion(c.clientSet)
	if err != nil {
		return nil, err
	}

	switch version {
	case kubernetes.StackAPIV1Beta1:
		return newStackV1Beta1(c.config, c.namespace)
	case kubernetes.StackAPIV1Beta2:
		return newStackV1Beta2(c.config, c.namespace)
	default:
		return nil, errors.Errorf("no supported Stack API version")
	}
}
