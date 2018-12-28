package v1beta2

import api "github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta2"

// ComposeV1beta2Interface defines the methods a compose v1beta2 client has
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta2.StackInterface instead
type ComposeV1beta2Interface = api.ComposeV1beta2Interface

// ComposeV1beta2Client is used to interact with features provided by the compose.docker.com group.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta2.ComposeV1beta2Client instead
type ComposeV1beta2Client = api.ComposeV1beta2Client

// NewForConfig creates a new ComposeV1beta2Client for the given config.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta2.NewForConfig instead
var NewForConfig = api.NewForConfig

// NewForConfigOrDie creates a new ComposeV1beta2Client for the given config and
// panics if there is an error in the config.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta2.NewForConfigOrDie instead
var NewForConfigOrDie = api.NewForConfigOrDie

// New creates a new ComposeV1beta2Client for the given RESTClient.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta2.New instead
var New = api.New
