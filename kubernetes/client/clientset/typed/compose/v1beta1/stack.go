package v1beta1

import api "github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta1"

// StacksGetter has a method to return a StackInterface.
// A group's client should implement this interface.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta1.StacksGetter instead
type StacksGetter = api.StacksGetter

// StackInterface has methods to work with Stack resources.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/clientset/typed/compose/v1beta1.StackInterface instead
type StackInterface = api.StackInterface
