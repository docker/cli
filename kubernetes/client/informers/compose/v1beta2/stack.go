package v1beta2

import (
	"time"

	"github.com/docker/cli/kubernetes/client/clientset"
	"github.com/docker/cli/kubernetes/client/informers/internalinterfaces"
	"github.com/docker/cli/kubernetes/client/listers/compose/v1beta2"
	compose_v1beta2 "github.com/docker/cli/kubernetes/compose/v1beta2"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// StackInformer provides access to a shared informer and lister for
// Stacks.
type StackInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1beta2.StackLister
}

type stackInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

func newStackInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	sharedIndexInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return client.ComposeV1beta2().Stacks(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.ComposeV1beta2().Stacks(v1.NamespaceAll).Watch(options)
			},
		},
		&compose_v1beta2.Stack{},
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	return sharedIndexInformer
}

func (f *stackInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&compose_v1beta2.Stack{}, newStackInformer)
}

func (f *stackInformer) Lister() v1beta2.StackLister {
	return v1beta2.NewStackLister(f.Informer().GetIndexer())
}
