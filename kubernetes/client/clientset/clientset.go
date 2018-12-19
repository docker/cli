package clientset

import api "github.com/docker/compose-on-kubernetes/api/client/clientset"

// Interface defines the methods a compose kube client should have
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset.Interface instead
type Interface = api.Interface

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset.Clientset instead
type Clientset = api.Clientset

// NewForConfig creates a new Clientset for the given config.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset.NewForConfig instead
var NewForConfig = api.NewForConfig

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset.NewForConfigOrDie instead
var NewForConfigOrDie = api.NewForConfigOrDie

// New creates a new Clientset for the given RESTClient.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset.New instead
var New = api.New
