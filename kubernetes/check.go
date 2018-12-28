package kubernetes

import api "github.com/docker/compose-on-kubernetes/api"

// StackVersion represents the detected Compose Component on Kubernetes side.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api.StackVersion instead
type StackVersion = api.StackVersion

const (
	// StackAPIV1Beta1 is returned if it's the most recent version available.
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api.StackAPIV1Beta1 instead
	StackAPIV1Beta1 = api.StackAPIV1Beta1
	// StackAPIV1Beta2 is returned if it's the most recent version available.
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api.StackAPIV1Beta2 instead
	StackAPIV1Beta2 = api.StackAPIV1Beta2
)

// GetStackAPIVersion returns the most recent stack API installed.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api.GetStackAPIVersion instead
var GetStackAPIVersion = api.GetStackAPIVersion
