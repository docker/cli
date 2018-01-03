package v1beta1

import (
	scheme "github.com/docker/cli/kubernetes/client/clientset_generated/clientset/scheme"
	v1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// StacksGetter has a method to return a StackInterface.
// A group's client should implement this interface.
type StacksGetter interface {
	Stacks(namespace string) StackInterface
}

// StackInterface has methods to work with Stack resources.
type StackInterface interface {
	Create(*v1beta1.Stack) (*v1beta1.Stack, error)
	Update(*v1beta1.Stack) (*v1beta1.Stack, error)
	UpdateStatus(*v1beta1.Stack) (*v1beta1.Stack, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1beta1.Stack, error)
	List(opts v1.ListOptions) (*v1beta1.StackList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.Stack, err error)
	StackExpansion
}

var _ StackInterface = &stacks{}

// stacks implements StackInterface
type stacks struct {
	client rest.Interface
	ns     string
}

// newStacks returns a Stacks
func newStacks(c *ComposeV1beta1Client, namespace string) *stacks {
	return &stacks{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Create takes the representation of a stack and creates it.  Returns the server's representation of the stack, and an error, if there is any.
func (c *stacks) Create(stack *v1beta1.Stack) (result *v1beta1.Stack, err error) {
	result = &v1beta1.Stack{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("stacks").
		Body(stack).
		Do().
		Into(result)
	return
}

// Update takes the representation of a stack and updates it. Returns the server's representation of the stack, and an error, if there is any.
func (c *stacks) Update(stack *v1beta1.Stack) (result *v1beta1.Stack, err error) {
	result = &v1beta1.Stack{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("stacks").
		Name(stack.Name).
		Body(stack).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclientstatus=false comment above the type to avoid generating UpdateStatus().

func (c *stacks) UpdateStatus(stack *v1beta1.Stack) (result *v1beta1.Stack, err error) {
	result = &v1beta1.Stack{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("stacks").
		Name(stack.Name).
		SubResource("status").
		Body(stack).
		Do().
		Into(result)
	return
}

// Delete takes name of the stack and deletes it. Returns an error if one occurs.
func (c *stacks) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("stacks").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *stacks) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("stacks").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the stack, and returns the corresponding stack object, and an error if there is any.
func (c *stacks) Get(name string, options v1.GetOptions) (result *v1beta1.Stack, err error) {
	result = &v1beta1.Stack{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("stacks").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Stacks that match those selectors.
func (c *stacks) List(opts v1.ListOptions) (result *v1beta1.StackList, err error) {
	result = &v1beta1.StackList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("stacks").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested stacks.
func (c *stacks) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("stacks").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched stack.
func (c *stacks) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.Stack, err error) {
	result = &v1beta1.Stack{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("stacks").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
