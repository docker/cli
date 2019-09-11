package v1beta1

import api "github.com/docker/compose-on-kubernetes/api/compose/v1beta1"

// GroupName is the group name used to register these objects
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.GroupName instead
const GroupName = api.GroupName

// Alias variables for the registration
var (
	// SchemeGroupVersion is group version used to register these objects
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.SchemeGroupVersion instead
	SchemeGroupVersion = api.SchemeGroupVersion
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.SchemeBuilder instead
	SchemeBuilder = api.SchemeBuilder
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.AddToScheme instead
	AddToScheme = api.AddToScheme
)

// Resource takes an unqualified resource and returns a Group qualified GroupResource
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.Resource instead
var Resource = api.Resource
