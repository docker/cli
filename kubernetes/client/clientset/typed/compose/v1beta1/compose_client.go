package v1beta1

import api "github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta1"

// ComposeV1beta1Interface defines the methods a compose v1beta1 client has
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta1.ComposeV1beta1Interface instead
type ComposeV1beta1Interface = api.ComposeV1beta1Interface

// ComposeV1beta1Client is used to interact with features provided by the compose.docker.com group.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta1.ComposeV1beta1Client instead
type ComposeV1beta1Client = api.ComposeV1beta1Client

// NewForConfig creates a new ComposeV1beta1Client for the given config.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta1.NewForConfig instead
var NewForConfig = api.NewForConfig

// NewForConfigOrDie creates a new ComposeV1beta1Client for the given config and
// panics if there is an error in the config.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta1.NewForConfigOrDie instead
var NewForConfigOrDie = api.NewForConfigOrDie

// New creates a new ComposeV1beta1Client for the given RESTClient.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta1.New instead
var New = api.New
