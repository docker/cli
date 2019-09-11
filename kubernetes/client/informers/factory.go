package informers

import api "github.com/docker/compose-on-kubernetes/api/client/informers"

// NewSharedInformerFactory constructs a new instance of sharedInformerFactory
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/informers.NewSharedInformerFactory instead
var NewSharedInformerFactory = api.NewSharedInformerFactory

// SharedInformerFactory provides shared informers for resources in all known
// API group versions.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/informers.SharedInformerFactory instead
type SharedInformerFactory = api.SharedInformerFactory
