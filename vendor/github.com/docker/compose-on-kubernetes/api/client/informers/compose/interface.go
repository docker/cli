package compose

import (
	"github.com/docker/compose-on-kubernetes/api/client/informers/compose/v1alpha3"
	"github.com/docker/compose-on-kubernetes/api/client/informers/compose/v1beta2"
	"github.com/docker/compose-on-kubernetes/api/client/informers/internalinterfaces"
)

// Interface provides access to each of this group's versions.
type Interface interface {
	V1beta2() v1beta2.Interface
	V1alpha3() v1alpha3.Interface
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

// V1alpha3 returns a new V1alpha3.Interface.
func (g *group) V1alpha3() v1alpha3.Interface {
	return v1alpha3.New(g.SharedInformerFactory)
}
