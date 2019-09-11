package scheme

import api "github.com/docker/compose-on-kubernetes/api/client/clientset/scheme"

// Variables required for registration
var (
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/scheme.Scheme instead
	Scheme = api.Scheme
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/scheme.Codecs instead
	Codecs = api.Codecs
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/scheme.ParameterCodec instead
	ParameterCodec = api.ParameterCodec
)

// AddToScheme adds all types of this clientset into the given scheme. This allows composition
// of clientsets, like in:
//
//   import (
//     "k8s.io/client-go/kubernetes"
//     clientsetscheme "k8s.io/client-go/kuberentes/scheme"
//     aggregatorclientsetscheme "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/scheme"
//   )
//
//   kclientset, _ := kubernetes.NewForConfig(c)
//   aggregatorclientsetscheme.AddToScheme(clientsetscheme.Scheme)
//
// After this, RawExtensions in Kubernetes types will serialize kube-aggregator types
// correctly.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/scheme.AddToScheme instead
var AddToScheme = api.AddToScheme
