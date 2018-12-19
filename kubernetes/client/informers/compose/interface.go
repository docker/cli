package compose

import api "github.com/docker/compose-on-kubernetes/api/client/informers/compose"

// Interface provides access to each of this group's versions.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/informers/compose.Interface instead
type Interface = api.Interface

// New returns a new Interface.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/client/informers/compose.New instead
var New = api.New
