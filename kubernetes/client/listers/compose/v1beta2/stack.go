package v1beta2

import api "github.com/docker/compose-on-kubernetes/api/client/listers/compose/v1beta2"

// StackLister helps list Stacks.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/listers/compose/v1beta2.StackLister instead
type StackLister = api.StackLister

// NewStackLister returns a new StackLister.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/listers/compose/v1beta2.NewStackLister instead
var NewStackLister = api.NewStackLister

// StackNamespaceLister helps list and get Stacks.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/listers/compose/v1beta2.StackNamespaceLister instead
type StackNamespaceLister = api.StackNamespaceLister
