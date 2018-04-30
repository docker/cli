package v1beta2

import (
	"github.com/docker/cli/kubernetes/compose/v1beta2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// StackLister helps list Stacks.
type StackLister interface {
	// List lists all Stacks in the indexer.
	List(selector labels.Selector) ([]*v1beta2.Stack, error)
	// Stacks returns an object that can list and get Stacks.
	Stacks(namespace string) StackNamespaceLister
	StackListerExpansion
}

// stackLister implements the StackLister interface.
type stackLister struct {
	indexer cache.Indexer
}

// NewStackLister returns a new StackLister.
func NewStackLister(indexer cache.Indexer) StackLister {
	return &stackLister{indexer: indexer}
}

// List lists all Stacks in the indexer.
func (s *stackLister) List(selector labels.Selector) ([]*v1beta2.Stack, error) {
	stacks := []*v1beta2.Stack{}
	err := cache.ListAll(s.indexer, selector, func(m interface{}) {
		stacks = append(stacks, m.(*v1beta2.Stack))
	})
	return stacks, err
}

// Stacks returns an object that can list and get Stacks.
func (s *stackLister) Stacks(namespace string) StackNamespaceLister {
	return stackNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// StackNamespaceLister helps list and get Stacks.
type StackNamespaceLister interface {
	// List lists all Stacks in the indexer for a given namespace.
	List(selector labels.Selector) ([]*v1beta2.Stack, error)
	// Get retrieves the Stack from the indexer for a given namespace and name.
	Get(name string) (*v1beta2.Stack, error)
	StackNamespaceListerExpansion
}

// stackNamespaceLister implements the StackNamespaceLister
// interface.
type stackNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Stacks in the indexer for a given namespace.
func (s stackNamespaceLister) List(selector labels.Selector) ([]*v1beta2.Stack, error) {
	stacks := []*v1beta2.Stack{}
	err := cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		stacks = append(stacks, m.(*v1beta2.Stack))
	})
	return stacks, err
}

// Get retrieves the Stack from the indexer for a given namespace and name.
func (s stackNamespaceLister) Get(name string) (*v1beta2.Stack, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1beta2.GroupResource("stack"), name)
	}
	return obj.(*v1beta2.Stack), nil
}
