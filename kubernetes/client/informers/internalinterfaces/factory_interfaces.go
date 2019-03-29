package internalinterfaces

import api "github.com/docker/compose-on-kubernetes/api/client/informers/internalinterfaces"

// NewInformerFunc defines a Informer constructor (from a clientset and a duration)
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/informers/internalinterfaces.NewInformerFunc instead
type NewInformerFunc = api.NewInformerFunc

// SharedInformerFactory a small interface to allow for adding an informer without an import cycle
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/informers/internalinterfaces.SharedInformerFactory instead
type SharedInformerFactory = api.SharedInformerFactory
