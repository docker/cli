package labels

import labels "github.com/docker/compose-on-kubernetes/api/labels"

const (
	// ForServiceName is the label for the service name.
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/labels.ForServiceName instead
	ForServiceName = labels.ForServiceName
	// ForStackName is the label for the stack name.
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/labels.ForStackName instead
	ForStackName = labels.ForStackName
	// ForServiceID is the label for the service id.
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/labels.ForServiceID instead
	ForServiceID = labels.ForServiceID
)

// ForService gives the labels to select a given service in a stack.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/labels.ForService instead
var ForService = labels.ForService

// SelectorForStack gives the labelSelector to use for a given stack.
// Specific service names can be passed to narrow down the selection.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/labels.SelectorForStack instead
var SelectorForStack = labels.SelectorForStack
