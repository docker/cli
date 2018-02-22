package compose

import (
	"github.com/docker/cli/kubernetes/client/informers/compose/v1beta2"
	"github.com/docker/cli/kubernetes/client/informers/internalinterfaces"
)

// Interface provides access to each of this group's versions.
type Interface interface {
	V1beta2() v1beta2.Interface
}

type group struct {
	internalinterfaces.SharedInformerFactory
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory) Interface {
	return &group{f}
}

// V1beta2 returns a new v1beta2.Interface.
func (g *group) V1beta2() v1beta2.Interface {
	return v1beta2.New(g.SharedInformerFactory)
}
