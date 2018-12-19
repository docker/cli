package v1beta2

import api "github.com/docker/compose-on-kubernetes/api/client/informers/compose/v1beta2"

// Interface provides access to all the informers in this group version.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/informers/compose/v1beta2.Interface instead
type Interface = api.Interface

// New returns a new Interface.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/informers/compose/v1beta2.New instead
var New = api.New
