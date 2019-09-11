package v1alpha3

import (
	"time"

	"github.com/docker/compose-on-kubernetes/api/client/clientset"
	"github.com/docker/compose-on-kubernetes/api/client/informers/internalinterfaces"
	"github.com/docker/compose-on-kubernetes/api/client/listers/compose/v1alpha3"
	compose_v1alpha3 "github.com/docker/compose-on-kubernetes/api/compose/v1alpha3"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// StackInformer provides access to a shared informer and lister for
// Stacks.
type StackInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha3.StackLister
}

type stackInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

func newStackInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	sharedIndexInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return client.ComposeV1alpha3().Stacks(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.ComposeV1alpha3().Stacks(v1.NamespaceAll).Watch(options)
			},
		},
		&compose_v1alpha3.Stack{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	return sharedIndexInformer
}

func (f *stackInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&compose_v1alpha3.Stack{}, newStackInformer)
}

func (f *stackInformer) Lister() v1alpha3.StackLister {
	return v1alpha3.NewStackLister(f.Informer().GetIndexer())
}

// NewFilteredStackInformer creates a stack informer with specific list options
func NewFilteredStackInformer(client clientset.Interface, resyncPeriod time.Duration, tweakListOptions func(*v1.ListOptions)) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ComposeV1alpha3().Stacks(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ComposeV1alpha3().Stacks(v1.NamespaceAll).Watch(options)
			},
		},
		&compose_v1alpha3.Stack{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}
