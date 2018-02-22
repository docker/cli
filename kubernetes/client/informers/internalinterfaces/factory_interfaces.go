package internalinterfaces

import (
	"time"

	"github.com/docker/cli/kubernetes/client/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// NewInformerFunc defines a Informer constructor (from a clientset and a duration)
type NewInformerFunc func(clientset.Interface, time.Duration) cache.SharedIndexInformer

// SharedInformerFactory a small interface to allow for adding an informer without an import cycle
type SharedInformerFactory interface {
	Start(stopCh <-chan struct{})
	InformerFor(obj runtime.Object, newFunc NewInformerFunc) cache.SharedIndexInformer
}
