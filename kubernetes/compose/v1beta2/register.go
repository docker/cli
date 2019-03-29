package v1beta2

import api "github.com/docker/compose-on-kubernetes/api/compose/v1beta2"

// GroupName is the name of the compose group
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.Owner instead
const GroupName = api.GroupName

var (
	// SchemeGroupVersion is group version used to register these objects
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.SchemeGroupVersion instead
	SchemeGroupVersion = api.SchemeGroupVersion
	// SchemeBuilder is the scheme builder
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.SchemeBuilder instead
	SchemeBuilder = api.SchemeBuilder
	// AddToScheme adds to scheme
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.AddToScheme instead
	AddToScheme = api.AddToScheme
)

// GroupResource takes an unqualified resource and returns a Group qualified GroupResource
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta2.GroupResource instead
var GroupResource = api.GroupResource
