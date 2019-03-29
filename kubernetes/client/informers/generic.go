package informers

import api "github.com/docker/compose-on-kubernetes/api/client/informers"

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/informers.GenericInformer instead
type GenericInformer = api.GenericInformer
