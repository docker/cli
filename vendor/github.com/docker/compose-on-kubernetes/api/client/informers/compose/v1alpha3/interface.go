package v1alpha3

import (
	"github.com/docker/compose-on-kubernetes/api/client/informers/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Stacks returns a StackInformer.
	Stacks() StackInformer
}

type version struct {
	internalinterfaces.SharedInformerFactory
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory) Interface {
	return &version{f}
}

// Stacks returns a StackInformer.
func (v *version) Stacks() StackInformer {
	return &stackInformer{factory: v.SharedInformerFactory}
}
