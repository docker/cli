package v1beta1

import api "github.com/docker/compose-on-kubernetes/api/compose/v1beta1"

// Owner defines the owner of a stack. It is used to impersonate the controller calls
// to kubernetes api.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.Owner instead
type Owner = api.Owner
